package service

import (
	"backend-go/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateJSONSubset(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"claimId": { "type": "string", "minLength": 1 },
			"count": { "type": "integer", "minimum": 0 },
			"status": { "enum": ["draft", "submitted"] },
			"tags": { "type": "array", "items": { "type": "string" } }
		},
		"required": ["claimId"]
	}`)

	valid := []byte(`{"claimId": "C-1", "count": 3, "status": "draft", "tags": ["a"]}`)
	if err := validateJSON(schema, valid); err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}

	cases := map[string]string{
		"missing required": `{"count": 1}`,
		"wrong type":       `{"claimId": 42}`,
		"empty minLength":  `{"claimId": ""}`,
		"bad enum":         `{"claimId": "C-1", "status": "approved"}`,
		"bad array item":   `{"claimId": "C-1", "tags": ["a", 5]}`,
		"not json":         `{oops`,
	}
	for name, input := range cases {
		if err := validateJSON(schema, []byte(input)); err == nil {
			t.Fatalf("case %q: expected validation error", name)
		}
	}

	if err := validateJSON(json.RawMessage(`{"unknownKeyword": 1}`), []byte(`"anything"`)); err != nil {
		t.Fatalf("unknown keywords must be ignored, got %v", err)
	}
}

// newExecFixture creates a provider + skill record backed by a real skill
// directory containing SKILL.md, package.json, and a scripts directory.
func newExecFixture(t *testing.T, manifest string, scriptName, scriptBody string) (*ExecService, *models.Skill, string) {
	t.Helper()
	db := openCatalogTestDB(t)
	root := t.TempDir()
	provider := createTestProvider(t, db, "Agents Global", root)

	skillRoot := filepath.Join(root, "fixture-skill")
	if err := os.MkdirAll(filepath.Join(skillRoot, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir fixture skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte("---\nname: fixture-skill\n---\n\nBody"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(skillRoot, ManifestFileName), []byte(manifest), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	}
	if scriptName != "" {
		scriptPath := filepath.Join(skillRoot, "scripts", scriptName)
		if err := os.WriteFile(scriptPath, []byte(scriptBody), 0o755); err != nil {
			t.Fatalf("write script: %v", err)
		}
	}

	skill := &models.Skill{
		ProviderID:    provider.ID,
		Name:          "fixture-skill",
		Slug:          "fixture-skill",
		DirectoryName: "fixture-skill",
		RootPath:      skillRoot,
		SkillMdPath:   filepath.Join(skillRoot, "SKILL.md"),
		Status:        "ready",
		Tags:          []string{},
		IssueCodes:    []string{},
		LastScannedAt: time.Now(),
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill record: %v", err)
	}
	return NewExecService(db), skill, skillRoot
}

const execFixtureManifest = `{
	"name": "fixture-skill",
	"version": "1.0.0",
	"scripts": {
		"hello": "bash scripts/hello.sh",
		"confirm-me": "bash scripts/hello.sh confirmed",
		"needs-env": "bash scripts/hello.sh",
		"read-stdin": "cat",
		"plain-args": "printf '%s\n' \"$@\"",
		"sleeps": "sleep 5",
		"fails": "exit 3"
	},
	"skm": {
		"schemaVersion": 1,
		"runtime": { "env": [] },
		"commands": {
			"confirm-me": { "confirm": "will do something final" },
			"needs-env": { "env": ["FIXTURE_KEY"] },
			"read-stdin": { "input": { "via": "stdin", "schema": { "type": "object", "required": ["id"] } } },
			"sleeps": { "timeoutSeconds": 1 }
		}
	}
}`

const helloScript = "#!/bin/sh\necho hello $*\n"

func TestExecRunsDeclaredCommandWithArgs(t *testing.T) {
	service, skill, skillRoot := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	var stdout bytes.Buffer
	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "plain-args",
		Args:     []string{"first", "second arg"},
		Stdout:   &stdout,
	})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if !result.OK || result.ExitCode != 0 {
		t.Fatalf("expected ok run, got %+v", result)
	}
	if result.WorkDir != skillRoot {
		t.Fatalf("expected workdir %q, got %q", skillRoot, result.WorkDir)
	}
	if stdout.String() != "first\nsecond arg\n" {
		t.Fatalf("unexpected stdout %q", stdout.String())
	}
	if got := result.Plan.FormatCommandLine(); !strings.Contains(got, "'second arg'") {
		t.Fatalf("expected quoted args in plan, got %q", got)
	}
}

func TestExecInjectsStandardEnv(t *testing.T) {
	manifest := `{
		"scripts": { "env-check": "bash scripts/env-check.sh" }
	}`
	script := "#!/bin/sh\necho root=$SKM_SKILL_ROOT zid=$SKM_SKILL_ZID cmd=$SKM_COMMAND custom=$FIXTURE_CUSTOM\n"
	service, skill, skillRoot := newExecFixture(t, manifest, "env-check.sh", script)

	var stdout bytes.Buffer
	_, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "env-check",
		Env:      []string{"FIXTURE_CUSTOM=injected"},
		Stdout:   &stdout,
	})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	expected := fmt.Sprintf("root=%s zid=%s cmd=env-check custom=injected\n", skillRoot, skill.Zid)
	if stdout.String() != expected {
		t.Fatalf("unexpected env output %q want %q", stdout.String(), expected)
	}
}

func TestExecCommandNotFoundListsAvailable(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	_, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "ghost"})
	var notFound *ExecCommandNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected ExecCommandNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "hello") || !strings.Contains(err.Error(), "plain-args") {
		t.Fatalf("expected available commands in error, got %q", err.Error())
	}
}

func TestExecRequiresManifest(t *testing.T) {
	service, skill, _ := newExecFixture(t, "", "", "")
	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "anything"}); !errors.Is(err, ErrExecManifestMissing) {
		t.Fatalf("expected ErrExecManifestMissing, got %v", err)
	}
}

func TestExecConfirmGate(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	_, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "confirm-me"})
	var confirmErr *ExecConfirmRequired
	if !errors.As(err, &confirmErr) {
		t.Fatalf("expected ExecConfirmRequired, got %v", err)
	}
	if !strings.Contains(confirmErr.Message, "will do something final") {
		t.Fatalf("expected manifest confirm message, got %q", confirmErr.Message)
	}

	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "confirm-me", AssumeYes: true})
	if err != nil || !result.OK {
		t.Fatalf("expected confirm-me to run with AssumeYes, err=%v result=%+v", err, result)
	}
}

func TestExecEnvGate(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	_, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "needs-env"})
	var missingErr *ExecMissingEnv
	if !errors.As(err, &missingErr) {
		t.Fatalf("expected ExecMissingEnv, got %v", err)
	}
	if len(missingErr.Missing) != 1 || missingErr.Missing[0] != "FIXTURE_KEY" {
		t.Fatalf("expected missing FIXTURE_KEY, got %v", missingErr.Missing)
	}

	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "needs-env",
		Env:      []string{"FIXTURE_KEY=value"},
	})
	if err != nil || !result.OK {
		t.Fatalf("expected needs-env to run with injected env, err=%v result=%+v", err, result)
	}
}

func TestExecStdinInputValidationAndDelivery(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	_, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid:  skill.Zid,
		Command:   "read-stdin",
		InputJSON: []byte(`{"wrong": true}`),
	})
	var inputErr *ExecInputInvalid
	if !errors.As(err, &inputErr) {
		t.Fatalf("expected ExecInputInvalid for schema violation, got %v", err)
	}

	var stdout bytes.Buffer
	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid:  skill.Zid,
		Command:   "read-stdin",
		InputJSON: []byte(`{"id": 7}`),
		Stdout:    &stdout,
	})
	if err != nil || !result.OK {
		t.Fatalf("expected stdin input to run, err=%v result=%+v", err, result)
	}
	if strings.TrimSpace(stdout.String()) != `{"id": 7}` {
		t.Fatalf("expected stdin delivered to script, got %q", stdout.String())
	}

	_, err = service.Exec(context.Background(), &ExecRequest{
		SkillZid:  skill.Zid,
		Command:   "hello",
		InputJSON: []byte(`{"id": 7}`),
	})
	if !errors.As(err, &inputErr) {
		t.Fatalf("expected ExecInputInvalid for undeclared input, got %v", err)
	}
}

func TestExecTimeoutKillsCommand(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	started := time.Now()
	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "sleeps"})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if !result.TimedOut || result.ExitCode != 124 {
		t.Fatalf("expected timed out with exit 124, got %+v", result)
	}
	if time.Since(started) > 4*time.Second {
		t.Fatalf("timeout did not kill promptly, took %v", time.Since(started))
	}
}

func TestExecPropagatesExitCode(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "fails"})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result.OK || result.ExitCode != 3 {
		t.Fatalf("expected exit code 3, got %+v", result)
	}
}

func TestExecDryRunDoesNotExecute(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "confirm-me",
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("dry run must bypass confirm gate, got %v", err)
	}
	if !result.DryRun || !result.OK {
		t.Fatalf("expected dry-run result, got %+v", result)
	}
	if result.Plan == nil || result.Plan.CommandLine != "bash scripts/hello.sh confirmed" {
		t.Fatalf("unexpected plan %+v", result.Plan)
	}
	if result.DurationMs != 0 || result.ExitCode != 0 {
		t.Fatalf("dry run must not execute, got %+v", result)
	}
}

func TestResolveExecDirPrefersRelationSource(t *testing.T) {
	source := t.TempDir()
	copyDir := t.TempDir()

	skill := &models.Skill{
		RootPath: copyDir,
		Relation: &models.SkillRelation{Mode: "from", FromPath: source},
	}
	if got := resolveExecDir(skill); got != source {
		t.Fatalf("expected source dir %q, got %q", source, got)
	}

	skill.Relation.FromPath = filepath.Join(t.TempDir(), "missing")
	if got := resolveExecDir(skill); got != copyDir {
		t.Fatalf("expected fallback to copy dir %q, got %q", copyDir, got)
	}
}
