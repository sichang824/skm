package service

import (
	"backend-go/internal/models"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// Exec cache layout: each skill gets ~/.skm/cache/exec/<zid>/ holding
//
//   - the materialized copy of the skill (only in cache mode),
//   - .skm-cache.json  — materialization metadata (freshness of the copy),
//   - .skm-setup.json  — the runtime.setup completion marker. The marker
//     lives in the cache directory even when execution happens in the source
//     directory, so source directories are never polluted by skm state.
const (
	cacheMetaFileName   = ".skm-cache.json"
	setupMarkerFileName = ".skm-setup.json"
)

// cacheMeta records where a materialized cache copy came from and whether it
// is still fresh, as described in the design doc (§4).
type cacheMeta struct {
	SourceZid      string    `json:"sourceZid"`
	SourcePath     string    `json:"sourcePath"`
	ContentHash    string    `json:"contentHash"`
	SourceHash     string    `json:"sourceHash"`
	MaterializedAt time.Time `json:"materializedAt"`
}

// setupMarkerFile records runtime.setup completions per working directory.
// An entry stays valid while the manifest content is unchanged; a changed
// package.json (new setup line, new dependency declarations) invalidates it
// and setup runs again. Source and cache-copy executions keep separate
// entries because they prepare different directories.
type setupMarkerFile struct {
	Entries map[string]setupMarkerEntry `json:"entries"` // key: working directory
}

type setupMarkerEntry struct {
	ManifestHash string    `json:"manifestHash"`
	SetupCommand string    `json:"setupCommand"`
	CompletedAt  time.Time `json:"completedAt"`
}

func defaultExecCacheRoot() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".skm", "cache", "exec")
	}
	return filepath.Join(homeDir, ".skm", "cache", "exec")
}

func (s *ExecService) cacheDirFor(zid string) string {
	return filepath.Join(s.cacheRoot, zid)
}

// execLocation is the resolved place where a command runs.
type execLocation struct {
	// WorkDir is the execution directory (source dir or cache copy).
	WorkDir string
	// ManifestDir is where package.json is read from. It differs from
	// WorkDir only for dry runs when the cache copy has not been
	// materialized yet (a dry run never writes).
	ManifestDir string
	// Mode is "source" or "cache".
	Mode string
	// CacheReused means an existing cache copy was fresh enough to keep.
	CacheReused bool
	// Materialized means the cache copy was (re)written by this call.
	Materialized bool
	// SourceHash is the tree hash of the content the cache copy holds
	// (empty in source mode; callers may compute it best-effort).
	SourceHash string
}

// resolveExecLocation implements design doc §4: source directory first,
// materialized cache for --isolate or when no local directory exists. A
// non-empty pin bypasses the normal rules (see resolvePinnedLocation).
func (s *ExecService) resolveExecLocation(skill *models.Skill, sourceDir string, sourceExists, isolate, dryRun bool, pin string) (*execLocation, error) {
	if pin != "" {
		return s.resolvePinnedLocation(skill, sourceDir, sourceExists, dryRun, pin)
	}
	if sourceExists && !isolate {
		return &execLocation{WorkDir: sourceDir, ManifestDir: sourceDir, Mode: "source"}, nil
	}

	cacheDir := s.cacheDirFor(skill.Zid)

	if sourceExists { // isolate with a local source
		rules := materializationRules(sourceDir)
		if dryRun {
			// Dry runs never write and never block on locks; predict reuse
			// best-effort from read-only state.
			if loc := reusableCacheLocation(skill, sourceDir, cacheDir, rules); loc != nil {
				return loc, nil
			}
			return &execLocation{WorkDir: cacheDir, ManifestDir: sourceDir, Mode: "cache"}, nil
		}
		// Writers hold the exclusive zid lock across check-then-materialize
		// so concurrent --isolate runs cannot wipe each other's copies.
		lock, err := s.acquireExecLock(skill.Zid, false)
		if err != nil {
			return nil, err
		}
		defer lock.release()
		if loc := reusableCacheLocation(skill, sourceDir, cacheDir, rules); loc != nil {
			return loc, nil
		}
		sourceHash, err := materializeSkill(sourceDir, cacheDir, rules)
		if err != nil {
			return nil, fmt.Errorf("materialize skill %s into cache: %w", skill.Zid, err)
		}
		meta := &cacheMeta{
			SourceZid:      skill.Zid,
			SourcePath:     sourceDir,
			ContentHash:    skill.ContentHash,
			SourceHash:     sourceHash,
			MaterializedAt: time.Now(),
		}
		if err := writeCacheMeta(cacheDir, meta); err != nil {
			return nil, fmt.Errorf("record cache metadata: %w", err)
		}
		return &execLocation{WorkDir: cacheDir, ManifestDir: cacheDir, Mode: "cache", Materialized: true, SourceHash: sourceHash}, nil
	}

	// No local directory. Remote/hub fetching does not exist yet; a
	// previously materialized copy whose contentHash still matches the
	// catalog record is the only executable fallback. Readers take the
	// shared lock so they never observe a half-materialized directory.
	if !dryRun {
		lock, err := s.acquireExecLock(skill.Zid, true)
		if err != nil {
			return nil, err
		}
		defer lock.release()
	}
	if meta := readCacheMeta(cacheDir); meta != nil && meta.ContentHash == skill.ContentHash && manifestExists(cacheDir) {
		return &execLocation{WorkDir: cacheDir, ManifestDir: cacheDir, Mode: "cache", CacheReused: true, SourceHash: meta.SourceHash}, nil
	}
	return nil, &ExecRootMissing{Path: sourceDir}
}

// reusableCacheLocation returns the cache location when the existing copy is
// fresh (contentHash matches the catalog record and the source tree is
// unchanged), otherwise nil.
func reusableCacheLocation(skill *models.Skill, sourceDir, cacheDir string, rules skillCopyRules) *execLocation {
	meta := readCacheMeta(cacheDir)
	if meta == nil || meta.ContentHash != skill.ContentHash || !manifestExists(cacheDir) {
		return nil
	}
	sourceHash, err := hashSkillTree(sourceDir, rules)
	if err != nil || sourceHash != meta.SourceHash {
		return nil
	}
	return &execLocation{WorkDir: cacheDir, ManifestDir: cacheDir, Mode: "cache", CacheReused: true, SourceHash: meta.SourceHash}
}

// resolvePinnedLocation resolves --pin (design doc §4): a cache copy whose
// source hash matches the pin wins; otherwise a matching source tree is
// materialized into the same (single) cache directory. Pinned execution
// never runs in the source directory, so replaying an old version cannot
// write runtime state into the live tree. When neither cache nor source
// matches, the version is unrecoverable and an error is returned.
func (s *ExecService) resolvePinnedLocation(skill *models.Skill, sourceDir string, sourceExists, dryRun bool, pin string) (*execLocation, error) {
	cacheDir := s.cacheDirFor(skill.Zid)

	if loc := pinnedCacheLocation(cacheDir, pin); loc != nil {
		return loc, nil
	}
	if sourceExists {
		rules := materializationRules(sourceDir)
		sourceHash, err := hashSkillTree(sourceDir, rules)
		if err == nil && strings.HasPrefix(sourceHash, pin) {
			if dryRun {
				// Report the plan without writing the cache copy.
				return &execLocation{WorkDir: cacheDir, ManifestDir: sourceDir, Mode: "cache", SourceHash: sourceHash}, nil
			}
			lock, err := s.acquireExecLock(skill.Zid, false)
			if err != nil {
				return nil, err
			}
			defer lock.release()
			// Re-check inside the lock; a concurrent run may have prepared
			// the pinned copy already.
			if loc := pinnedCacheLocation(cacheDir, pin); loc != nil {
				return loc, nil
			}
			if _, err := materializeSkill(sourceDir, cacheDir, rules); err != nil {
				return nil, fmt.Errorf("materialize skill %s into cache: %w", skill.Zid, err)
			}
			meta := &cacheMeta{
				SourceZid:      skill.Zid,
				SourcePath:     sourceDir,
				ContentHash:    skill.ContentHash,
				SourceHash:     sourceHash,
				MaterializedAt: time.Now(),
			}
			if err := writeCacheMeta(cacheDir, meta); err != nil {
				return nil, fmt.Errorf("record cache metadata: %w", err)
			}
			return &execLocation{WorkDir: cacheDir, ManifestDir: cacheDir, Mode: "cache", Materialized: true, SourceHash: sourceHash}, nil
		}
	}
	return nil, &ExecPinUnavailable{SkillZid: skill.Zid, Pin: pin}
}

// pinnedCacheLocation returns the cache location when the cached copy's
// source hash matches the pin. The ContentHash equality check is bypassed on
// purpose: the pin names the exact content to run.
func pinnedCacheLocation(cacheDir, pin string) *execLocation {
	meta := readCacheMeta(cacheDir)
	if meta != nil && strings.HasPrefix(meta.SourceHash, pin) && manifestExists(cacheDir) {
		return &execLocation{WorkDir: cacheDir, ManifestDir: cacheDir, Mode: "cache", CacheReused: true, SourceHash: meta.SourceHash}
	}
	return nil
}

// execLock serializes cache operations for one skill. The lock file is a
// SIBLING of the cache directory (<cacheRoot>/<zid>.lock) because
// materializeSkill wipes the cache directory itself.
type execLock struct {
	file *os.File
}

// acquireExecLock takes an advisory lock for one skill's cache operations:
// shared for read-only paths, exclusive for check-run-write sections
// (materialization, dependency installation, setup). The lock is never held
// while the actual command runs.
func (s *ExecService) acquireExecLock(zid string, shared bool) (*execLock, error) {
	if err := os.MkdirAll(s.cacheRoot, 0o755); err != nil {
		return nil, fmt.Errorf("prepare exec cache root: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(s.cacheRoot, zid+".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open exec cache lock: %w", err)
	}
	how := syscall.LOCK_EX
	if shared {
		how = syscall.LOCK_SH
	}
	if err := syscall.Flock(int(file.Fd()), how); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock exec cache for %s: %w", zid, err)
	}
	return &execLock{file: file}, nil
}

func (l *execLock) release() {
	if l == nil || l.file == nil {
		return
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	_ = l.file.Close()
}

func manifestExists(dir string) bool {
	info, err := os.Stat(ManifestPath(dir))
	return err == nil && !info.IsDir()
}

func isExistingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// forcedManifestFiles are always part of a cache copy: the manifest itself
// and the dependency lockfiles that managed installs (skm.runtime.deps) rely
// on. Without forcing them, .to-narrowed skills would lose their lockfiles
// in cache copies.
var forcedManifestFiles = []string{ManifestFileName, packageLockFileName, requirementsFileName}

// materializationRules decides which files a cache copy contains. When the
// source directory has .to metadata, its include/exclude rules define the
// portable content of the skill (as the design doc prescribes); manifest
// and lockfiles are always force-included because exec needs them. Without
// .to, everything is copied except well-known junk.
func materializationRules(sourceDir string) skillCopyRules {
	state, err := readSkillRelationState(sourceDir)
	if err == nil && state.HasTo {
		rules := copyRulesFromMetadata(state.To)
		rules.Include = ensureIncluded(rules.Include, forcedManifestFiles)
		return rules
	}
	return skillCopyRules{
		Include: []string{"**"},
		Exclude: []string{
			"**/.DS_Store",
			"**/.git",
			"**/.git/**",
			"**/node_modules/**",
			"**/__pycache__/**",
			"**/.venv/**",
		},
	}
}

// ensureIncluded prepends any forced file names not already covered by the
// include patterns.
func ensureIncluded(include []string, forced []string) []string {
	if len(include) == 0 {
		return append([]string{}, forced...)
	}
	missing := make([]string, 0, len(forced))
	for _, name := range forced {
		covered := false
		for _, pattern := range include {
			if pattern == name || pattern == "**" || pattern == "**/*" || pattern == "*" {
				covered = true
				break
			}
		}
		if !covered {
			missing = append(missing, name)
		}
	}
	return append(missing, include...)
}

// junkDirectoryNames are never walked during hashing or materialization,
// regardless of copy rules. This keeps cache operations fast for skills that
// carry heavy local state.
var junkDirectoryNames = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"__pycache__":  {},
	".venv":        {},
}

// hashSkillTree hashes all files of a directory that the copy rules would
// copy. The hash covers relative paths and file contents, so any change that
// would alter a materialized copy changes the hash.
func hashSkillTree(root string, rules skillCopyRules) (string, error) {
	entries := make([]string, 0, 64)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		name := entry.Name()
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if _, junk := junkDirectoryNames[name]; junk {
				return filepath.SkipDir
			}
			return nil
		}
		shouldCopy, err := rules.shouldCopy(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		if !shouldCopy {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entries = append(entries, filepath.ToSlash(rel)+"\x00symlink:"+target)
			return nil
		}
		sum, err := hashFileContents(path)
		if err != nil {
			return err
		}
		entries = append(entries, filepath.ToSlash(rel)+"\x00"+sum)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(entries)
	sum := sha256.Sum256([]byte(strings.Join(entries, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func hashFileContents(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// materializeSkill replaces the cache directory with a fresh copy of the
// source directory, applying the copy rules. It returns the source hash of
// the copied content.
func materializeSkill(sourceDir, cacheDir string, rules skillCopyRules) (string, error) {
	if err := os.RemoveAll(cacheDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}

	sourceHash, err := hashSkillTree(sourceDir, rules)
	if err != nil {
		return "", err
	}

	err = filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceDir {
			return nil
		}
		name := entry.Name()
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if _, junk := junkDirectoryNames[name]; junk {
				return filepath.SkipDir
			}
			return nil
		}
		shouldCopy, err := rules.shouldCopy(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		if !shouldCopy {
			return nil
		}
		targetPath := filepath.Join(cacheDir, rel)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, targetPath)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, targetPath, info.Mode().Perm())
	})
	if err != nil {
		return "", err
	}
	return sourceHash, nil
}

func readCacheMeta(cacheDir string) *cacheMeta {
	data, err := os.ReadFile(filepath.Join(cacheDir, cacheMetaFileName))
	if err != nil {
		return nil
	}
	var meta cacheMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil
	}
	return &meta
}

func writeCacheMeta(cacheDir string, meta *cacheMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, cacheMetaFileName), append(data, '\n'), 0o644)
}

func readSetupMarker(cacheDir string) *setupMarkerFile {
	data, err := os.ReadFile(filepath.Join(cacheDir, setupMarkerFileName))
	if err != nil {
		return nil
	}
	var marker setupMarkerFile
	if err := json.Unmarshal(data, &marker); err != nil {
		return nil
	}
	if marker.Entries == nil {
		marker.Entries = map[string]setupMarkerEntry{}
	}
	return &marker
}

func writeSetupMarker(cacheDir string, marker *setupMarkerFile) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, setupMarkerFileName), append(data, '\n'), 0o644)
}

// readSetupEntry returns the recorded setup completion for one working
// directory, if any.
func readSetupEntry(cacheDir, workDir string) *setupMarkerEntry {
	marker := readSetupMarker(cacheDir)
	if marker == nil {
		return nil
	}
	if entry, ok := marker.Entries[workDir]; ok {
		return &entry
	}
	return nil
}

func writeSetupEntry(cacheDir, workDir string, entry setupMarkerEntry) error {
	marker := readSetupMarker(cacheDir)
	if marker == nil {
		marker = &setupMarkerFile{Entries: map[string]setupMarkerEntry{}}
	}
	marker.Entries[workDir] = entry
	return writeSetupMarker(cacheDir, marker)
}

// setupEntryFresh reports whether a recorded setup completion still applies
// to the current manifest content.
func setupEntryFresh(entry *setupMarkerEntry, manifestHash string) bool {
	return entry != nil && entry.ManifestHash == manifestHash
}
