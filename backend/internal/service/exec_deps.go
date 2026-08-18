package service

import (
	"backend-go/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Managed dependency installation (design doc §4 and §11.1). Skills opt in
// by declaring skm.runtime.deps in package.json; skm then runs the
// ecosystem's standard installer before runtime.setup:
//
//   - node:   package-lock.json present -> npm ci, otherwise npm install
//   - python: requirements.txt -> python venv (.venv) + pip install
//
// Completion markers (.skm-deps.json) bind to the lockfile/requirements
// content hashes per working directory, so installs are idempotent and
// re-run only when the declarations change. Rematerializing a cache copy
// wipes the marker along with the directory, so isolated copies reinstall —
// node_modules and .venv never travel in cache copies.
const depsMarkerFileName = ".skm-deps.json"

// Well-known dependency declaration files. They are force-included in cache
// copies (see materializationRules) and are the hash anchors of install
// idempotency.
const (
	packageLockFileName  = "package-lock.json"
	requirementsFileName = "requirements.txt"
)

const (
	npmCiCommand      = "npm ci --no-audit --no-fund"
	npmInstallCommand = "npm install --no-audit --no-fund"
	venvCreateCommand = "python3 -m venv .venv"
	pipInstallCommand = ".venv/bin/pip install -r requirements.txt"
)

// depsMarkerFile records managed dependency installation completions per
// working directory, mirroring the setup marker layout.
type depsMarkerFile struct {
	Entries map[string]depsMarkerEntry `json:"entries"` // key: working directory
}

type depsMarkerEntry struct {
	NodeHash    string    `json:"nodeHash"`
	PythonHash  string    `json:"pythonHash"`
	CompletedAt time.Time `json:"completedAt"`
}

// DepsInfo describes what managed dependency installation did during one
// invocation: it ran (Ran), was skipped because the marker is still fresh
// (Skipped), or failed (non-zero ExitCode / TimedOut).
type DepsInfo struct {
	Node   string `json:"node,omitempty"`   // node command line that ran
	Python string `json:"python,omitempty"` // python command line that ran
	Ran    bool   `json:"ran,omitempty"`
	// Skipped is true when the marker is fresh; a nil *DepsInfo means the
	// skill declares no managed deps at all.
	Skipped    bool   `json:"skipped,omitempty"`
	ExitCode   int    `json:"exitCode,omitempty"`
	TimedOut   bool   `json:"timedOut,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Output     string `json:"output,omitempty"`
}

// ensureDeps installs managed dependencies when needed and maintains the
// completion marker. It returns nil when the skill declares no managed deps.
// Concurrent installs serialize on the exclusive zid lock; the freshness
// check repeats inside the lock so only one of them installs.
func (s *ExecService) ensureDeps(ctx context.Context, skill *models.Skill, manifest *ParsedManifest, workDir string, stdout, stderr io.Writer) (*DepsInfo, error) {
	nodeManaged, pythonManaged := manifest.Deps.Resolve(
		fileExists(filepath.Join(workDir, packageLockFileName)),
		manifest.HasNPMDependencies,
		fileExists(filepath.Join(workDir, requirementsFileName)),
	)
	if !nodeManaged && !pythonManaged {
		return nil, nil
	}

	nodeHash := depsNodeHash(workDir, manifest)
	pythonHash := depsPythonHash(workDir)
	cacheDir := s.cacheDirFor(skill.Zid)
	if depsEntryFresh(readDepsEntry(cacheDir, workDir), nodeHash, pythonHash) {
		return &DepsInfo{Skipped: true}, nil
	}

	lock, err := s.acquireExecLock(skill.Zid, false)
	if err != nil {
		return nil, err
	}
	defer lock.release()
	if depsEntryFresh(readDepsEntry(cacheDir, workDir), nodeHash, pythonHash) {
		return &DepsInfo{Skipped: true}, nil
	}

	info := &DepsInfo{}
	env := buildExecEnv(workDir, skill, manifest, "deps", nil, "", nil)

	if nodeManaged {
		line := npmInstallCommand
		if fileExists(filepath.Join(workDir, packageLockFileName)) {
			line = npmCiCommand
		}
		info.Node = line
		step, err := runLifecycleCommand(ctx, workDir, line, env, 0, stdout, stderr)
		if err != nil {
			return nil, fmt.Errorf("start node dependency install: %w", err)
		}
		info.Ran = true
		info.ExitCode = step.ExitCode
		info.TimedOut = step.TimedOut
		info.DurationMs += step.DurationMs
		info.Output += step.Output
		if step.ExitCode != 0 || step.TimedOut {
			return info, nil
		}
	}

	if pythonManaged {
		lines := make([]string, 0, 2)
		if !isExistingDir(filepath.Join(workDir, ".venv")) {
			step, err := runLifecycleCommand(ctx, workDir, venvCreateCommand, env, 0, stdout, stderr)
			if err != nil {
				return nil, fmt.Errorf("start python venv creation: %w", err)
			}
			info.Ran = true
			info.DurationMs += step.DurationMs
			info.Output += step.Output
			lines = append(lines, venvCreateCommand)
			if step.ExitCode != 0 || step.TimedOut {
				info.ExitCode = step.ExitCode
				info.TimedOut = step.TimedOut
				info.Python = strings.Join(append(lines, pipInstallCommand), " && ")
				return info, nil
			}
		}
		step, err := runLifecycleCommand(ctx, workDir, pipInstallCommand, env, 0, stdout, stderr)
		if err != nil {
			return nil, fmt.Errorf("start pip install: %w", err)
		}
		info.Ran = true
		info.DurationMs += step.DurationMs
		info.Output += step.Output
		lines = append(lines, pipInstallCommand)
		info.Python = strings.Join(lines, " && ")
		info.ExitCode = step.ExitCode
		info.TimedOut = step.TimedOut
		if step.ExitCode != 0 || step.TimedOut {
			return info, nil
		}
	}

	if err := writeDepsEntry(cacheDir, workDir, depsMarkerEntry{
		NodeHash:    nodeHash,
		PythonHash:  pythonHash,
		CompletedAt: time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("record dependency installation: %w", err)
	}
	return info, nil
}

// depsPlan predicts the dependency actions of a run without acquiring locks
// or writing anything (dry-run support). It returns the planned action
// lines, or skipped=true when the marker is already fresh.
func (s *ExecService) depsPlan(manifest *ParsedManifest, cacheDir, workDir string) (actions []string, skipped bool) {
	nodeManaged, pythonManaged := manifest.Deps.Resolve(
		fileExists(filepath.Join(workDir, packageLockFileName)),
		manifest.HasNPMDependencies,
		fileExists(filepath.Join(workDir, requirementsFileName)),
	)
	if !nodeManaged && !pythonManaged {
		return nil, false
	}
	if depsEntryFresh(readDepsEntry(cacheDir, workDir), depsNodeHash(workDir, manifest), depsPythonHash(workDir)) {
		return nil, true
	}
	if nodeManaged {
		line := npmInstallCommand
		if fileExists(filepath.Join(workDir, packageLockFileName)) {
			line = npmCiCommand
		}
		actions = append(actions, "node: "+line)
	}
	if pythonManaged {
		line := pipInstallCommand
		if !isExistingDir(filepath.Join(workDir, ".venv")) {
			line = venvCreateCommand + " && " + pipInstallCommand
		}
		actions = append(actions, "python: "+line)
	}
	return actions, false
}

// depsNodeHash is the idempotency key of the node install step: the lockfile
// content hash when present, otherwise the manifest hash (dependency
// declarations live in package.json), or empty when neither exists.
func depsNodeHash(workDir string, manifest *ParsedManifest) string {
	if sum, err := hashFileContents(filepath.Join(workDir, packageLockFileName)); err == nil {
		return sum
	}
	if manifest.HasNPMDependencies {
		return manifest.Hash()
	}
	return ""
}

// depsPythonHash is the idempotency key of the python install step (empty
// when no requirements.txt exists).
func depsPythonHash(workDir string) string {
	sum, err := hashFileContents(filepath.Join(workDir, requirementsFileName))
	if err != nil {
		return ""
	}
	return sum
}

// depsEntryFresh reports whether a recorded installation still applies to
// the current dependency declarations.
func depsEntryFresh(entry *depsMarkerEntry, nodeHash, pythonHash string) bool {
	return entry != nil && entry.NodeHash == nodeHash && entry.PythonHash == pythonHash
}

func readDepsMarker(cacheDir string) *depsMarkerFile {
	data, err := os.ReadFile(filepath.Join(cacheDir, depsMarkerFileName))
	if err != nil {
		return nil
	}
	var marker depsMarkerFile
	if err := json.Unmarshal(data, &marker); err != nil {
		return nil
	}
	if marker.Entries == nil {
		marker.Entries = map[string]depsMarkerEntry{}
	}
	return &marker
}

// readDepsEntry returns the recorded installation for one working directory,
// if any.
func readDepsEntry(cacheDir, workDir string) *depsMarkerEntry {
	marker := readDepsMarker(cacheDir)
	if marker == nil {
		return nil
	}
	if entry, ok := marker.Entries[workDir]; ok {
		return &entry
	}
	return nil
}

func writeDepsEntry(cacheDir, workDir string, entry depsMarkerEntry) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	marker := readDepsMarker(cacheDir)
	if marker == nil {
		marker = &depsMarkerFile{Entries: map[string]depsMarkerEntry{}}
	}
	marker.Entries[workDir] = entry
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, depsMarkerFileName), append(data, '\n'), 0o644)
}

// fileExists reports whether path exists as a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
