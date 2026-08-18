package service

import (
	"backend-go/internal/models"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newCacheFixture is newExecFixture with the exec cache redirected into a
// temp directory so cache tests never touch ~/.skm.
func newCacheFixture(t *testing.T, manifest string, files map[string]string) (*ExecService, *models.Skill, string) {
	t.Helper()
	service, skill, skillRoot := newExecFixture(t, manifest, "", "")
	service.cacheRoot = t.TempDir()
	for relPath, content := range files {
		target := filepath.Join(skillRoot, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir for fixture file %s: %v", relPath, err)
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(relPath, ".sh") {
			mode = 0o755
		}
		if err := os.WriteFile(target, []byte(content), mode); err != nil {
			t.Fatalf("write fixture file %s: %v", relPath, err)
		}
	}
	return service, skill, skillRoot
}

const isolateManifest = `{
	"name": "fixture-skill",
	"version": "1.0.0",
	"scripts": { "toucher": "bash scripts/toucher.sh" }
}`

const toucherScript = "#!/bin/sh\ntouch ran-here.txt\necho done\n"

func TestExecIsolateRunsInCacheAndProtectsSource(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})

	var stdout bytes.Buffer
	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Isolate:  true,
		Stdout:   &stdout,
	})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if !result.OK || result.ExitCode != 0 {
		t.Fatalf("expected ok isolated run, got %+v", result)
	}

	cacheDir := service.cacheDirFor(skill.Zid)
	if result.WorkDir != cacheDir {
		t.Fatalf("expected cache workdir %q, got %q", cacheDir, result.WorkDir)
	}
	if result.Plan.Mode != "cache" || !result.Plan.Materialized {
		t.Fatalf("expected materialized cache plan, got %+v", result.Plan)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "ran-here.txt")); err != nil {
		t.Fatalf("expected command side effect in cache copy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(skillRoot, "ran-here.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source directory must stay clean, stat got %v", err)
	}
	for _, name := range []string{"SKILL.md", ManifestFileName, filepath.Join("scripts", "toucher.sh")} {
		if _, err := os.Stat(filepath.Join(cacheDir, name)); err != nil {
			t.Fatalf("cache copy missing %s: %v", name, err)
		}
	}
	if meta := readCacheMeta(cacheDir); meta == nil || meta.SourcePath != skillRoot {
		t.Fatalf("expected cache metadata with source path, got %+v", meta)
	}
}

func TestExecIsolateReusesFreshCache(t *testing.T) {
	service, skill, _ := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})

	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher", Isolate: true}); err != nil {
		t.Fatalf("first isolated run: %v", err)
	}
	cacheDir := service.cacheDirFor(skill.Zid)
	sentinel := filepath.Join(cacheDir, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("kept"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher", Isolate: true})
	if err != nil {
		t.Fatalf("second isolated run: %v", err)
	}
	if !result.Plan.CacheReused || result.Plan.Materialized {
		t.Fatalf("expected cache reuse, got %+v", result.Plan)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("reused cache must keep previous state: %v", err)
	}
}

func TestExecIsolateRematerializesWhenSourceChanges(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})

	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher", Isolate: true}); err != nil {
		t.Fatalf("first isolated run: %v", err)
	}
	cacheDir := service.cacheDirFor(skill.Zid)
	sentinel := filepath.Join(cacheDir, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	updated := "#!/bin/sh\ntouch ran-here.txt\necho updated\n"
	if err := os.WriteFile(filepath.Join(skillRoot, "scripts", "toucher.sh"), []byte(updated), 0o755); err != nil {
		t.Fatalf("update script: %v", err)
	}

	var stdout bytes.Buffer
	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Isolate:  true,
		Stdout:   &stdout,
	})
	if err != nil {
		t.Fatalf("second isolated run: %v", err)
	}
	if result.Plan.CacheReused || !result.Plan.Materialized {
		t.Fatalf("expected rematerialization after source change, got %+v", result.Plan)
	}
	if strings.TrimSpace(stdout.String()) != "updated" {
		t.Fatalf("expected updated script output, got %q", stdout.String())
	}
	if _, err := os.Stat(sentinel); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rematerialized cache must be a clean copy, stat got %v", err)
	}
}

func TestExecDryRunIsolateDoesNotWriteCache(t *testing.T) {
	service, skill, _ := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})

	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Isolate:  true,
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("dry run returned error: %v", err)
	}
	cacheDir := service.cacheDirFor(skill.Zid)
	if _, err := os.Stat(cacheDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry run must not materialize, stat got %v", err)
	}
	if result.Plan.Mode != "cache" || result.Plan.WorkDir != cacheDir {
		t.Fatalf("expected cache plan pointing at %q, got %+v", cacheDir, result.Plan)
	}
	if result.Plan.CommandLine != "bash scripts/toucher.sh" {
		t.Fatalf("expected plan resolved from source manifest, got %+v", result.Plan)
	}
}

func TestMaterializationHonorsToRulesAndKeepsManifest(t *testing.T) {
	toMeta := `{
		"directories": [],
		"include": ["SKILL.md", "scripts/**"],
		"exclude": ["**/.DS_Store"]
	}`
	service, skill, _ := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"media/blob.txt":     "heavy state",
		"notes.txt":          "not distributed",
		".to":                toMeta,
	})

	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher", Isolate: true}); err != nil {
		t.Fatalf("isolated run: %v", err)
	}
	cacheDir := service.cacheDirFor(skill.Zid)

	for _, kept := range []string{"SKILL.md", ManifestFileName, filepath.Join("scripts", "toucher.sh")} {
		if _, err := os.Stat(filepath.Join(cacheDir, kept)); err != nil {
			t.Fatalf("cache copy missing %s: %v", kept, err)
		}
	}
	for _, dropped := range []string{filepath.Join("media", "blob.txt"), "notes.txt", ".to"} {
		if _, err := os.Stat(filepath.Join(cacheDir, dropped)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("cache copy must not contain %s, stat got %v", dropped, err)
		}
	}
}

func TestCacheCopyServesExecAfterSourceDisappears(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	skill.ContentHash = "fixed-hash"
	if err := service.catalog.db.Save(skill).Error; err != nil {
		t.Fatalf("save skill content hash: %v", err)
	}

	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher", Isolate: true}); err != nil {
		t.Fatalf("materializing run: %v", err)
	}
	if err := os.RemoveAll(skillRoot); err != nil {
		t.Fatalf("remove source dir: %v", err)
	}

	var stdout bytes.Buffer
	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Stdout:   &stdout,
	})
	if err != nil {
		t.Fatalf("exec after source removal: %v", err)
	}
	if !result.Plan.CacheReused || result.Plan.Mode != "cache" {
		t.Fatalf("expected cache fallback, got %+v", result.Plan)
	}
	if strings.TrimSpace(stdout.String()) != "done" {
		t.Fatalf("expected cached copy to run, got %q", stdout.String())
	}
}

func TestExecRootMissingWithoutCache(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	if err := os.RemoveAll(skillRoot); err != nil {
		t.Fatalf("remove source dir: %v", err)
	}
	_, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher"})
	var missing *ExecRootMissing
	if !errors.As(err, &missing) {
		t.Fatalf("expected ExecRootMissing, got %v", err)
	}
}

// --- runtime.setup idempotency ---

const setupManifest = `{
	"name": "fixture-skill",
	"version": "1.0.0",
	"scripts": {
		"prep": "bash scripts/prep.sh",
		"hello": "bash scripts/hello.sh"
	},
	"skm": {
		"schemaVersion": 1,
		"runtime": { "setup": "prep" }
	}
}`

const prepScript = "#!/bin/sh\necho run >> setup-runs.txt\n"

func TestExecSetupRunsOnceThenSkips(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, setupManifest, map[string]string{
		"scripts/prep.sh":  prepScript,
		"scripts/hello.sh": helloScript,
	})

	first, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello"})
	if err != nil || !first.OK {
		t.Fatalf("first run: err=%v result=%+v", err, first)
	}
	if first.Setup == nil || !first.Setup.Ran || first.Setup.Skipped {
		t.Fatalf("expected setup to run on first exec, got %+v", first.Setup)
	}
	assertLineCount(t, filepath.Join(skillRoot, "setup-runs.txt"), 1)

	second, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello"})
	if err != nil || !second.OK {
		t.Fatalf("second run: err=%v result=%+v", err, second)
	}
	if second.Setup == nil || !second.Setup.Skipped || second.Setup.Ran {
		t.Fatalf("expected setup to be skipped on second exec, got %+v", second.Setup)
	}
	assertLineCount(t, filepath.Join(skillRoot, "setup-runs.txt"), 1)

	if second.Plan == nil || second.Plan.Setup != "prep" || !second.Plan.SetupSkipped {
		t.Fatalf("expected plan to report fresh setup marker, got %+v", second.Plan)
	}
}

func TestExecSetupRerunsWhenManifestChanges(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, setupManifest, map[string]string{
		"scripts/prep.sh":  prepScript,
		"scripts/hello.sh": helloScript,
	})

	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello"}); err != nil {
		t.Fatalf("first run: %v", err)
	}
	updatedManifest := strings.Replace(setupManifest, `"1.0.0"`, `"1.1.0"`, 1)
	if err := os.WriteFile(filepath.Join(skillRoot, ManifestFileName), []byte(updatedManifest), 0o644); err != nil {
		t.Fatalf("update manifest: %v", err)
	}

	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello"})
	if err != nil || !result.OK {
		t.Fatalf("run after manifest change: err=%v result=%+v", err, result)
	}
	if result.Setup == nil || !result.Setup.Ran {
		t.Fatalf("expected setup re-run after manifest change, got %+v", result.Setup)
	}
	assertLineCount(t, filepath.Join(skillRoot, "setup-runs.txt"), 2)
}

func TestExecSetupFailureAbortsCommand(t *testing.T) {
	failingManifest := `{
		"scripts": {
			"prep": "bash scripts/prep.sh",
			"hello": "bash scripts/hello.sh"
		},
		"skm": { "schemaVersion": 1, "runtime": { "setup": "prep" } }
	}`
	service, skill, skillRoot := newCacheFixture(t, failingManifest, map[string]string{
		"scripts/prep.sh":  "#!/bin/sh\necho boom >&2\nexit 5\n",
		"scripts/hello.sh": "#!/bin/sh\ntouch hello-ran.txt\necho hello\n",
	})

	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello"})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result.OK || result.ExitCode != 5 || result.Aborted != "setup-failed" {
		t.Fatalf("expected setup-failed abort with exit 5, got %+v", result)
	}
	if _, err := os.Stat(filepath.Join(skillRoot, "hello-ran.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("command must not run after setup failure, stat got %v", err)
	}

	// A failed setup must not leave a completion marker behind.
	if marker := readSetupMarker(service.cacheDirFor(skill.Zid)); marker != nil {
		t.Fatalf("failed setup must not write a marker, got %+v", marker)
	}
}

func TestExecSetupKeepsSeparateStateForIsolatedRuns(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, setupManifest, map[string]string{
		"scripts/prep.sh":  prepScript,
		"scripts/hello.sh": helloScript,
	})

	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello"}); err != nil {
		t.Fatalf("source run: %v", err)
	}
	assertLineCount(t, filepath.Join(skillRoot, "setup-runs.txt"), 1)

	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello", Isolate: true})
	if err != nil || !result.OK {
		t.Fatalf("isolated run: err=%v result=%+v", err, result)
	}
	if result.Setup == nil || !result.Setup.Ran {
		t.Fatalf("cache copy needs its own setup run, got %+v", result.Setup)
	}
	// The cache copy snapshots source state (1 line) and the cache-local
	// setup appends its own line; the source file stays untouched.
	assertLineCount(t, filepath.Join(result.WorkDir, "setup-runs.txt"), 2)
	assertLineCount(t, filepath.Join(skillRoot, "setup-runs.txt"), 1)
}

func TestRunSetupExplicitModes(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, setupManifest, map[string]string{
		"scripts/prep.sh":  prepScript,
		"scripts/hello.sh": helloScript,
	})

	dry, err := service.RunSetup(context.Background(), &SetupRequest{SkillZid: skill.Zid, DryRun: true})
	if err != nil || !dry.DryRun || !dry.OK {
		t.Fatalf("dry run setup: err=%v result=%+v", err, dry)
	}
	if dry.Plan.CommandLine != "bash scripts/prep.sh" || dry.Plan.SetupSkipped {
		t.Fatalf("unexpected dry setup plan: %+v", dry.Plan)
	}
	assertLineCount(t, filepath.Join(skillRoot, "setup-runs.txt"), 0)

	first, err := service.RunSetup(context.Background(), &SetupRequest{SkillZid: skill.Zid})
	if err != nil || !first.OK {
		t.Fatalf("explicit setup: err=%v result=%+v", err, first)
	}
	if first.Setup == nil || !first.Setup.Ran {
		t.Fatalf("expected setup to run, got %+v", first.Setup)
	}
	assertLineCount(t, filepath.Join(skillRoot, "setup-runs.txt"), 1)

	second, err := service.RunSetup(context.Background(), &SetupRequest{SkillZid: skill.Zid})
	if err != nil || !second.OK {
		t.Fatalf("repeat setup: err=%v result=%+v", err, second)
	}
	if second.Setup == nil || !second.Setup.Skipped || second.Setup.Ran {
		t.Fatalf("expected idempotent skip, got %+v", second.Setup)
	}
	assertLineCount(t, filepath.Join(skillRoot, "setup-runs.txt"), 1)

	forced, err := service.RunSetup(context.Background(), &SetupRequest{SkillZid: skill.Zid, Force: true})
	if err != nil || !forced.OK {
		t.Fatalf("forced setup: err=%v result=%+v", err, forced)
	}
	if forced.Setup == nil || !forced.Setup.Ran {
		t.Fatalf("expected forced re-run, got %+v", forced.Setup)
	}
	assertLineCount(t, filepath.Join(skillRoot, "setup-runs.txt"), 2)
}

func TestRunSetupMissingDeclaration(t *testing.T) {
	service, skill, _ := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	if _, err := service.RunSetup(context.Background(), &SetupRequest{SkillZid: skill.Zid}); !errors.Is(err, ErrExecSetupMissing) {
		t.Fatalf("expected ErrExecSetupMissing, got %v", err)
	}
}

func TestSetupTimeoutFromAnnotation(t *testing.T) {
	manifest := `{
		"scripts": {
			"prep": "sleep 5",
			"hello": "bash scripts/hello.sh"
		},
		"skm": {
			"schemaVersion": 1,
			"runtime": { "setup": "prep" },
			"commands": { "prep": { "timeoutSeconds": 1 } }
		}
	}`
	service, skill, _ := newCacheFixture(t, manifest, map[string]string{
		"scripts/hello.sh": helloScript,
	})

	started := time.Now()
	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello"})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result.Aborted != "setup-failed" || !result.Setup.TimedOut || result.ExitCode != 124 {
		t.Fatalf("expected timed-out setup abort, got %+v", result)
	}
	if time.Since(started) > 4*time.Second {
		t.Fatalf("setup timeout did not kill promptly, took %v", time.Since(started))
	}
}

func assertLineCount(t *testing.T, path string, expected int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if expected != 0 {
				t.Fatalf("expected %d lines in %s, file missing", expected, path)
			}
			return
		}
		t.Fatalf("read %s: %v", path, err)
	}
	count := len(strings.Split(strings.TrimSpace(string(data)), "\n"))
	if strings.TrimSpace(string(data)) == "" {
		count = 0
	}
	if count != expected {
		t.Fatalf("expected %d lines in %s, got %d (%q)", expected, path, count, string(data))
	}
}
