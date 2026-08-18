package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Dependency-install tests shim npm/python3 via PATH so no real toolchain
// runs. Each shim appends "<tool> <args>" to a shared call log; the python3
// shim also materializes .venv/bin/pip (itself a shim) on `python3 -m venv`.

// depsFixture is a cache fixture with shimmed tools and a call log.
type depsFixture struct {
	service *ExecService
	skill   *skillWithRoot
	logPath string
}

type skillWithRoot struct {
	zid  string
	root string
}

func newDepsFixture(t *testing.T, manifest string, files map[string]string, npmExit int) *depsFixture {
	t.Helper()
	service, skill, skillRoot := newCacheFixture(t, manifest, files)

	shimDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "calls.log")
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	writeNpmShim(t, shimDir, logPath, npmExit)
	writePythonShim(t, shimDir, logPath)

	return &depsFixture{
		service: service,
		skill:   &skillWithRoot{zid: skill.Zid, root: skillRoot},
		logPath: logPath,
	}
}

func writeNpmShim(t *testing.T, shimDir, logPath string, exitCode int) {
	t.Helper()
	body := fmt.Sprintf("#!/bin/sh\necho \"npm $*\" >> '%s'\nexit %d\n", logPath, exitCode)
	if err := os.WriteFile(filepath.Join(shimDir, "npm"), []byte(body), 0o755); err != nil {
		t.Fatalf("write npm shim: %v", err)
	}
}

func writePythonShim(t *testing.T, shimDir, logPath string) {
	t.Helper()
	body := `#!/bin/sh
echo "python3 $*" >> '` + logPath + `'
if [ "$1" = "-m" ] && [ "$2" = "venv" ]; then
  mkdir -p "$3/bin"
  cat > "$3/bin/pip" <<'EOF'
#!/bin/sh
echo "pip $*" >> '` + logPath + `'
exit 0
EOF
  chmod +x "$3/bin/pip"
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(shimDir, "python3"), []byte(body), 0o755); err != nil {
		t.Fatalf("write python3 shim: %v", err)
	}
}

func (f *depsFixture) calls(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(f.logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		t.Fatalf("read call log: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func (f *depsFixture) callCount(t *testing.T, prefix string) int {
	t.Helper()
	count := 0
	for _, call := range f.calls(t) {
		if strings.HasPrefix(call, prefix) {
			count++
		}
	}
	return count
}

func (f *depsFixture) execToucher(t *testing.T) *ExecResult {
	t.Helper()
	result, err := f.service.Exec(context.Background(), &ExecRequest{
		SkillZid: f.skill.zid,
		Command:  "toucher",
	})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	return result
}

const depsAutoManifest = `{
	"name": "fixture-skill",
	"version": "1.0.0",
	"scripts": { "toucher": "bash scripts/toucher.sh" },
	"skm": { "schemaVersion": 1, "runtime": { "deps": {} } }
}`

const depsNodeOnlyManifest = `{
	"name": "fixture-skill",
	"version": "1.0.0",
	"dependencies": { "left-pad": "1.3.0" },
	"scripts": { "toucher": "bash scripts/toucher.sh" },
	"skm": { "schemaVersion": 1, "runtime": { "deps": { "node": true } } }
}`

const depsPythonOnlyManifest = `{
	"name": "fixture-skill",
	"version": "1.0.0",
	"scripts": { "toucher": "bash scripts/toucher.sh" },
	"skm": { "schemaVersion": 1, "runtime": { "deps": { "python": true } } }
}`

func TestDepsLockfileRunsNpmCiOnce(t *testing.T) {
	fixture := newDepsFixture(t, depsAutoManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"package-lock.json":  "{\n  \"name\": \"fixture-skill\",\n  \"lockfileVersion\": 3\n}\n",
	}, 0)

	result := fixture.execToucher(t)
	if !result.OK {
		t.Fatalf("expected ok run, got %+v", result)
	}
	if result.Deps == nil || !result.Deps.Ran || result.Deps.Node != npmCiCommand {
		t.Fatalf("expected npm ci to run, got %+v", result.Deps)
	}
	if got := fixture.callCount(t, "npm ci"); got != 1 {
		t.Fatalf("npm ci called %d times, want 1", got)
	}

	// Second run: marker fresh, install skipped.
	second := fixture.execToucher(t)
	if second.Deps == nil || !second.Deps.Skipped {
		t.Fatalf("expected deps skip on second run, got %+v", second.Deps)
	}
	if got := fixture.callCount(t, "npm ci"); got != 1 {
		t.Fatalf("npm ci called %d times after idempotent rerun, want 1", got)
	}
}

func TestDepsNoLockfileRunsNpmInstall(t *testing.T) {
	fixture := newDepsFixture(t, depsNodeOnlyManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	}, 0)

	result := fixture.execToucher(t)
	if result.Deps == nil || result.Deps.Node != npmInstallCommand {
		t.Fatalf("expected npm install without lockfile, got %+v", result.Deps)
	}
	if got := fixture.callCount(t, "npm install"); got != 1 {
		t.Fatalf("npm install called %d times, want 1", got)
	}
}

func TestDepsUndeclaredNeverInstalls(t *testing.T) {
	fixture := newDepsFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"package-lock.json":  "{}",
	}, 0)

	result := fixture.execToucher(t)
	if !result.OK {
		t.Fatalf("expected ok run, got %+v", result)
	}
	if result.Deps != nil {
		t.Fatalf("undeclared deps must not produce DepsInfo, got %+v", result.Deps)
	}
	if calls := fixture.calls(t); len(calls) != 0 {
		t.Fatalf("undeclared deps must not call installers, got %v", calls)
	}
	if entry := readDepsEntry(fixture.service.cacheDirFor(fixture.skill.zid), fixture.skill.root); entry != nil {
		t.Fatalf("undeclared deps must not write a marker, got %+v", entry)
	}
}

func TestDepsPythonCreatesVenvAndInjectsPath(t *testing.T) {
	const manifest = `{
		"name": "fixture-skill",
		"version": "1.0.0",
		"scripts": {
			"toucher": "bash scripts/toucher.sh",
			"show-path": "echo PATH=$PATH"
		},
		"skm": { "schemaVersion": 1, "runtime": { "deps": { "python": true } } }
	}`
	fixture := newDepsFixture(t, manifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"requirements.txt":   "flask==3.0.0\n",
	}, 0)

	result := fixture.execToucher(t)
	if result.Deps == nil || !result.Deps.Ran {
		t.Fatalf("expected python deps to run, got %+v", result.Deps)
	}
	want := venvCreateCommand + " && " + pipInstallCommand
	if result.Deps.Python != want {
		t.Fatalf("python step mismatch:\n got %q\nwant %q", result.Deps.Python, want)
	}
	if fixture.callCount(t, "python3 -m venv") != 1 || fixture.callCount(t, "pip install -r requirements.txt") != 1 {
		t.Fatalf("expected one venv creation and one pip install, got %v", fixture.calls(t))
	}

	// The skill-local venv bin must lead PATH so scripts find its python.
	var stdout strings.Builder
	var stderr strings.Builder
	pathResult, err := fixture.service.Exec(context.Background(), &ExecRequest{
		SkillZid: fixture.skill.zid,
		Command:  "show-path",
		Stdout:   &stdout,
		Stderr:   &stderr,
	})
	if err != nil || !pathResult.OK {
		t.Fatalf("show-path failed: %+v / %v", pathResult, err)
	}
	venvBin := filepath.Join(fixture.skill.root, ".venv", "bin")
	if !strings.Contains(stdout.String(), "PATH="+venvBin+string(os.PathListSeparator)) {
		t.Fatalf("PATH must start with %s, got %q", venvBin, stdout.String())
	}
}

func TestDepsExistingVenvSkipsCreation(t *testing.T) {
	fixture := newDepsFixture(t, depsPythonOnlyManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"requirements.txt":   "flask==3.0.0\n",
	}, 0)
	// Pre-existing venv with its own logging pip shim.
	pipDir := filepath.Join(fixture.skill.root, ".venv", "bin")
	if err := os.MkdirAll(pipDir, 0o755); err != nil {
		t.Fatalf("mkdir venv bin: %v", err)
	}
	pipBody := fmt.Sprintf("#!/bin/sh\necho \"pip $*\" >> '%s'\nexit 0\n", fixture.logPath)
	if err := os.WriteFile(filepath.Join(pipDir, "pip"), []byte(pipBody), 0o755); err != nil {
		t.Fatalf("write pip shim: %v", err)
	}

	fixture.execToucher(t)
	if got := fixture.callCount(t, "python3"); got != 0 {
		t.Fatalf("existing .venv must not trigger venv creation, python3 called %d times", got)
	}
	if got := fixture.callCount(t, "pip install"); got != 1 {
		t.Fatalf("pip install called %d times, want 1", got)
	}
}

func TestDepsFailureAbortsCommand(t *testing.T) {
	fixture := newDepsFixture(t, depsAutoManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"package-lock.json":  "{}",
	}, 7)

	result, err := fixture.service.Exec(context.Background(), &ExecRequest{
		SkillZid: fixture.skill.zid,
		Command:  "toucher",
	})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result.Aborted != "deps-failed" || result.ExitCode != 7 {
		t.Fatalf("expected deps-failed abort with exit 7, got %+v", result)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.skill.root, "ran-here.txt")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("command must not start after a deps failure, stat got %v", statErr)
	}
	if entry := readDepsEntry(fixture.service.cacheDirFor(fixture.skill.zid), fixture.skill.root); entry != nil {
		t.Fatalf("failed install must not write a marker, got %+v", entry)
	}
}

func TestDepsRequirementsChangeReinstalls(t *testing.T) {
	fixture := newDepsFixture(t, depsPythonOnlyManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"requirements.txt":   "flask==3.0.0\n",
	}, 0)

	fixture.execToucher(t)
	if err := os.WriteFile(filepath.Join(fixture.skill.root, "requirements.txt"), []byte("flask==3.1.0\n"), 0o644); err != nil {
		t.Fatalf("update requirements: %v", err)
	}
	fixture.execToucher(t)

	if got := fixture.callCount(t, "pip install"); got != 2 {
		t.Fatalf("pip install called %d times after requirements change, want 2", got)
	}
}

func TestDepsIsolateRematerializeReinstalls(t *testing.T) {
	fixture := newDepsFixture(t, depsAutoManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"package-lock.json":  "{}",
	}, 0)

	if _, err := fixture.service.Exec(context.Background(), &ExecRequest{
		SkillZid: fixture.skill.zid,
		Command:  "toucher",
		Isolate:  true,
	}); err != nil {
		t.Fatalf("first isolated run: %v", err)
	}
	if got := fixture.callCount(t, "npm ci"); got != 1 {
		t.Fatalf("npm ci called %d times after first isolated run, want 1", got)
	}

	// Change the source so rematerialization wipes the cache marker.
	if err := os.WriteFile(filepath.Join(fixture.skill.root, "extra.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("write extra file: %v", err)
	}
	if _, err := fixture.service.Exec(context.Background(), &ExecRequest{
		SkillZid: fixture.skill.zid,
		Command:  "toucher",
		Isolate:  true,
	}); err != nil {
		t.Fatalf("second isolated run: %v", err)
	}
	if got := fixture.callCount(t, "npm ci"); got != 2 {
		t.Fatalf("npm ci called %d times after rematerialize, want 2", got)
	}
}

func TestDepsToRulesStillShipLockfiles(t *testing.T) {
	manifest := `{
		"name": "fixture-skill",
		"version": "1.0.0",
		"scripts": { "toucher": "echo cache-ok > ran-here.txt" },
		"skm": { "schemaVersion": 1, "runtime": { "deps": {} } }
	}`
	fixture := newDepsFixture(t, manifest, map[string]string{
		"package-lock.json": "{}",
		"requirements.txt":  "flask==3.0.0\n",
	}, 0)
	// Narrow .to rules: the forced manifest/lockfile set must still ship.
	toBody := `{"include": ["SKILL.md"]}`
	if err := os.WriteFile(filepath.Join(fixture.skill.root, ".to"), []byte(toBody), 0o644); err != nil {
		t.Fatalf("write .to: %v", err)
	}

	result, err := fixture.service.Exec(context.Background(), &ExecRequest{
		SkillZid: fixture.skill.zid,
		Command:  "toucher",
		Isolate:  true,
	})
	if err != nil || !result.OK {
		t.Fatalf("isolated run with narrow .to failed: %+v / %v", result, err)
	}

	cacheDir := fixture.service.cacheDirFor(fixture.skill.zid)
	for _, name := range []string{ManifestFileName, "package-lock.json", "requirements.txt", "SKILL.md"} {
		if _, err := os.Stat(filepath.Join(cacheDir, name)); err != nil {
			t.Fatalf("cache copy missing forced file %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(cacheDir, ".to")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("narrow .to rules must not ship arbitrary files, stat got %v", err)
	}
}

func TestDepsDryRunPlansWithoutInstalling(t *testing.T) {
	fixture := newDepsFixture(t, depsAutoManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"package-lock.json":  "{}",
	}, 0)

	result, err := fixture.service.Exec(context.Background(), &ExecRequest{
		SkillZid: fixture.skill.zid,
		Command:  "toucher",
		DryRun:   true,
	})
	if err != nil || !result.DryRun {
		t.Fatalf("dry run failed: %+v / %v", result, err)
	}
	if len(result.Plan.Deps) == 0 || !strings.Contains(result.Plan.Deps[0], npmCiCommand) {
		t.Fatalf("dry-run plan should list the node install, got %+v", result.Plan.Deps)
	}
	if calls := fixture.calls(t); len(calls) != 0 {
		t.Fatalf("dry run must not call installers, got %v", calls)
	}
}

func TestRunSetupRunsDepsFirst(t *testing.T) {
	const manifest = `{
		"name": "fixture-skill",
		"version": "1.0.0",
		"scripts": {
			"toucher": "bash scripts/toucher.sh",
			"setup": "echo setup-ran >> setup-order.txt"
		},
		"skm": { "schemaVersion": 1, "runtime": { "deps": {}, "setup": "setup" } }
	}`
	fixture := newDepsFixture(t, manifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"package-lock.json":  "{}",
	}, 0)

	result, err := fixture.service.RunSetup(context.Background(), &SetupRequest{SkillZid: fixture.skill.zid})
	if err != nil || !result.OK {
		t.Fatalf("RunSetup failed: %+v / %v", result, err)
	}
	if result.Deps == nil || !result.Deps.Ran {
		t.Fatalf("RunSetup must install managed deps, got %+v", result.Deps)
	}
	if _, err := os.Stat(filepath.Join(fixture.skill.root, "setup-order.txt")); err != nil {
		t.Fatalf("setup command did not run after deps: %v", err)
	}
}
