package service

import (
	"backend-go/internal/models"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- flock (design doc §11.2) ---

func TestExecIsolateCreatesSiblingLockFile(t *testing.T) {
	service, skill, _ := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})

	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher", Isolate: true}); err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}

	// The lock file must be a sibling of the cache directory: materialize
	// wipes the directory itself, so a lock inside it would not survive.
	lockPath := filepath.Join(service.cacheRoot, skill.Zid+".lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected sibling lock file %s: %v", lockPath, err)
	}
}

func TestAcquireExecLockSerializesWriters(t *testing.T) {
	service, skill, _ := newCacheFixture(t, isolateManifest, nil)

	first, err := service.acquireExecLock(skill.Zid, false)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}

	acquired := make(chan struct{})
	go func() {
		second, err := service.acquireExecLock(skill.Zid, false)
		if err == nil {
			second.release()
		}
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("second writer acquired the lock while the first writer held it")
	case <-time.After(150 * time.Millisecond):
		// expected: the second writer blocks
	}

	first.release()
	select {
	case <-acquired:
	case <-time.After(3 * time.Second):
		t.Fatal("second writer never acquired the lock after release")
	}
}

func TestConcurrentIsolateExecsAllSucceed(t *testing.T) {
	service, skill, _ := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})

	const writers = 4
	var wg sync.WaitGroup
	errs := make([]error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := service.Exec(context.Background(), &ExecRequest{
				SkillZid: skill.Zid,
				Command:  "toucher",
				Isolate:  true,
			})
			if err == nil && (!result.OK || result.ExitCode != 0) {
				err = fmt.Errorf("run %d: unexpected result %+v", index, result)
			}
			errs[index] = err
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	if meta := readCacheMeta(service.cacheDirFor(skill.Zid)); meta == nil || meta.SourceHash == "" {
		t.Fatal("expected one valid cache meta after concurrent isolate runs")
	}
}

func TestConcurrentFirstRunsExecuteSetupOnce(t *testing.T) {
	const manifest = `{
		"name": "fixture-skill",
		"version": "1.0.0",
		"scripts": {
			"hello": "bash scripts/hello.sh",
			"count": "bash scripts/count.sh"
		},
		"skm": { "schemaVersion": 1, "runtime": { "setup": "count" } }
	}`
	service, skill, skillRoot := newExecFixture(t, manifest, "hello.sh", helloScript)
	service.cacheRoot = t.TempDir()
	countScript := "#!/bin/sh\necho run >> setup-count.txt\n"
	if err := os.WriteFile(filepath.Join(skillRoot, "scripts", "count.sh"), []byte(countScript), 0o755); err != nil {
		t.Fatalf("write count script: %v", err)
	}

	const runners = 4
	var wg sync.WaitGroup
	for i := 0; i < runners; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello"}); err != nil {
				t.Errorf("concurrent exec: %v", err)
			}
		}()
	}
	wg.Wait()

	data, err := os.ReadFile(filepath.Join(skillRoot, "setup-count.txt"))
	if err != nil {
		t.Fatalf("read setup counter: %v", err)
	}
	if runs := strings.Count(string(data), "run"); runs != 1 {
		t.Fatalf("setup ran %d times under concurrency, want exactly 1", runs)
	}
}

// --- version pin (design doc §11.5) ---

func materializeOnce(t *testing.T, service *ExecService, skill *models.Skill) *ExecResult {
	t.Helper()
	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Isolate:  true,
	})
	if err != nil {
		t.Fatalf("materialize run: %v", err)
	}
	return result
}

func TestExecPinReusesMatchingCacheCopy(t *testing.T) {
	service, skill, _ := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	materializeOnce(t, service, skill)

	cacheDir := service.cacheDirFor(skill.Zid)
	meta := readCacheMeta(cacheDir)
	if meta == nil {
		t.Fatal("expected cache meta after materialization")
	}
	// A sentinel in the cache copy proves reuse: rematerialization wipes it.
	if err := os.WriteFile(filepath.Join(cacheDir, "sentinel.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Pin:      meta.SourceHash[:12],
	})
	if err != nil {
		t.Fatalf("pinned exec: %v", err)
	}
	if result.WorkDir != cacheDir || !result.Plan.CacheReused {
		t.Fatalf("expected pinned run to reuse the cache copy, got %+v", result.Plan)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "sentinel.txt")); err != nil {
		t.Fatalf("pinned run must reuse the existing copy, sentinel lost: %v", err)
	}
}

func TestExecPinRematerializesFromMatchingSource(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	materializeOnce(t, service, skill)

	// Evolve the source to v2 and pin the new version.
	if err := os.WriteFile(filepath.Join(skillRoot, "extra.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("write extra file: %v", err)
	}
	v2, err := hashSkillTree(skillRoot, materializationRules(skillRoot))
	if err != nil {
		t.Fatalf("hash source tree: %v", err)
	}

	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Pin:      v2[:16],
	})
	if err != nil {
		t.Fatalf("pinned exec: %v", err)
	}
	if !result.Plan.Materialized {
		t.Fatalf("expected rematerialization for the pinned source version, got %+v", result.Plan)
	}
	if meta := readCacheMeta(service.cacheDirFor(skill.Zid)); meta == nil || meta.SourceHash != v2 {
		t.Fatalf("cache meta should record the pinned version hash, got %+v", meta)
	}
}

func TestExecPinNeverRunsInSourceDirectory(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	sourceHash, err := hashSkillTree(skillRoot, materializationRules(skillRoot))
	if err != nil {
		t.Fatalf("hash source tree: %v", err)
	}

	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Pin:      sourceHash[:12],
	})
	if err != nil {
		t.Fatalf("pinned exec: %v", err)
	}
	if result.Plan.Mode != "cache" {
		t.Fatalf("pinned runs must use a cache copy, got mode %q", result.Plan.Mode)
	}
	if _, err := os.Stat(filepath.Join(skillRoot, "ran-here.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pinned run must not write into the source directory, stat got %v", err)
	}
}

func TestExecPinUnavailable(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})

	_, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Pin:      "deadbeef00000000",
	})
	var unavailable *ExecPinUnavailable
	if !errors.As(err, &unavailable) {
		t.Fatalf("expected ExecPinUnavailable, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(skillRoot, "ran-here.txt")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("nothing must run when the pin is unavailable, stat got %v", statErr)
	}

	// Dry runs surface the same error.
	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher", Pin: "deadbeef00000000", DryRun: true}); !errors.As(err, &unavailable) {
		t.Fatalf("expected ExecPinUnavailable on dry run, got %v", err)
	}
}

func TestExecPinInvalidFormat(t *testing.T) {
	service, skill, _ := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	for _, pin := range []string{"xyz", "abcd", "deadbeefzzzz"} {
		_, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher", Pin: pin})
		var invalid *ExecPinInvalid
		if !errors.As(err, &invalid) {
			t.Fatalf("pin %q: expected ExecPinInvalid, got %v", pin, err)
		}
	}
}

func TestExecPinAcceptsFullHash(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	sourceHash, err := hashSkillTree(skillRoot, materializationRules(skillRoot))
	if err != nil {
		t.Fatalf("hash source tree: %v", err)
	}
	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Pin:      sourceHash, // all 64 hex chars
	})
	if err != nil || !result.OK {
		t.Fatalf("full-hash pin should resolve, got %+v / %v", result, err)
	}
}

func TestExecPinDryRunDoesNotWrite(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	sourceHash, err := hashSkillTree(skillRoot, materializationRules(skillRoot))
	if err != nil {
		t.Fatalf("hash source tree: %v", err)
	}

	cacheDir := service.cacheDirFor(skill.Zid)
	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Pin:      sourceHash[:12],
		DryRun:   true,
	})
	if err != nil || !result.DryRun {
		t.Fatalf("dry-run pinned exec failed: %+v / %v", result, err)
	}
	if result.Plan.WorkDir != cacheDir || result.Plan.Pin != sourceHash[:12] {
		t.Fatalf("dry-run plan should point at the cache dir with the pin, got %+v", result.Plan)
	}
	if _, statErr := os.Stat(cacheDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("dry run must not create the cache directory, stat got %v", statErr)
	}
}

func TestExecPinWithSourceGone(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	materializeOnce(t, service, skill)
	meta := readCacheMeta(service.cacheDirFor(skill.Zid))
	if meta == nil {
		t.Fatal("expected cache meta after materialization")
	}

	if err := os.RemoveAll(skillRoot); err != nil {
		t.Fatalf("remove source dir: %v", err)
	}

	// The pinned copy in the cache still runs.
	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Pin:      meta.SourceHash[:12],
	})
	if err != nil || !result.OK {
		t.Fatalf("pinned run from cache after source removal failed: %+v / %v", result, err)
	}

	// A non-matching pin is unrecoverable.
	var unavailable *ExecPinUnavailable
	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "toucher", Pin: "deadbeef00000000"}); !errors.As(err, &unavailable) {
		t.Fatalf("expected ExecPinUnavailable with source gone, got %v", err)
	}
}

func TestExecPinSurvivesContentHashDrift(t *testing.T) {
	service, skill, _ := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	materializeOnce(t, service, skill)
	cacheDir := service.cacheDirFor(skill.Zid)
	meta := readCacheMeta(cacheDir)
	if meta == nil {
		t.Fatal("expected cache meta after materialization")
	}

	// The catalog hash drifts (e.g. SKILL.md edited); the pin bypasses the
	// ContentHash equality check by design.
	skill.ContentHash = "drifted-content-hash"
	if err := service.catalog.db.Save(skill).Error; err != nil {
		t.Fatalf("save drifted hash: %v", err)
	}

	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Pin:      meta.SourceHash[:12],
	})
	if err != nil || !result.OK || !result.Plan.CacheReused {
		t.Fatalf("pin should reuse the cache copy despite ContentHash drift, got %+v / %v", result, err)
	}
}
