package service

import (
	"backend-go/internal/models"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkillFixture(t *testing.T, skillMd, manifest string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(root, ManifestFileName), []byte(manifest), 0o644); err != nil {
			t.Fatalf("write %s: %v", ManifestFileName, err)
		}
	}
	return root
}

func TestLoadManifestMissingReturnsSentinel(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadManifest(root); err != ErrManifestNotFound {
		t.Fatalf("expected ErrManifestNotFound, got %v", err)
	}
}

func TestParseManifestFullAnnotation(t *testing.T) {
	manifest, err := ParseManifest([]byte(`{
		"name": "tingwu-transcribe",
		"version": "1.2.0",
		"scripts": {
			"setup": "bash scripts/setup.sh",
			"transcribe": "bash scripts/transcribe.sh",
			"list-media": "bash scripts/list_media.sh"
		},
		"skm": {
			"schemaVersion": 1,
			"runtime": { "env": ["TINGWU_API_KEY", "TINGWU_API_KEY", " "], "setup": "setup" },
			"commands": {
				"transcribe": {
					"description": "Transcribe a local audio file",
					"confirm": true,
					"timeoutSeconds": 1800,
					"env": ["TINGWU_DIR_ID"],
					"input": { "via": "argv", "schema": { "type": "object" } }
				},
				"list-media": { "description": "List media files" }
			}
		}
	}`))
	if err != nil {
		t.Fatalf("ParseManifest returned error: %v", err)
	}
	if manifest.Name != "tingwu-transcribe" || manifest.Version != "1.2.0" {
		t.Fatalf("unexpected identity name=%q version=%q", manifest.Name, manifest.Version)
	}
	if manifest.SchemaVersion != 1 {
		t.Fatalf("expected schemaVersion=1, got %d", manifest.SchemaVersion)
	}
	if len(manifest.RuntimeEnv) != 1 || manifest.RuntimeEnv[0] != "TINGWU_API_KEY" {
		t.Fatalf("expected deduped runtime env [TINGWU_API_KEY], got %v", manifest.RuntimeEnv)
	}
	if manifest.SetupCommand != "setup" {
		t.Fatalf("expected setup command, got %q", manifest.SetupCommand)
	}
	if len(manifest.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(manifest.Commands))
	}
	if manifest.Commands[0].Name != "list-media" || manifest.Commands[1].Name != "setup" || manifest.Commands[2].Name != "transcribe" {
		t.Fatalf("expected commands sorted by name, got %v", manifest.Commands)
	}

	var transcribe *models.SkillCommand
	for index := range manifest.Commands {
		if manifest.Commands[index].Name == "transcribe" {
			transcribe = &manifest.Commands[index]
		}
	}
	if transcribe == nil {
		t.Fatal("transcribe command not found")
	}
	if transcribe.Description != "Transcribe a local audio file" {
		t.Fatalf("unexpected description %q", transcribe.Description)
	}
	if transcribe.Confirm != "requires confirmation before running" {
		t.Fatalf("expected confirm=true to normalize to default message, got %q", transcribe.Confirm)
	}
	if transcribe.TimeoutSeconds != 1800 {
		t.Fatalf("expected timeoutSeconds=1800, got %d", transcribe.TimeoutSeconds)
	}
	if len(transcribe.Env) != 1 || transcribe.Env[0] != "TINGWU_DIR_ID" {
		t.Fatalf("unexpected command env %v", transcribe.Env)
	}
	if transcribe.InputVia != "argv" || !transcribe.HasInputSchema {
		t.Fatalf("expected argv input with schema, got via=%q hasSchema=%t", transcribe.InputVia, transcribe.HasInputSchema)
	}
}

func TestParseManifestConfirmStringAndDefaultInputVia(t *testing.T) {
	manifest, err := ParseManifest([]byte(`{
		"scripts": { "delete": "node run.js delete" },
		"skm": {
			"commands": {
				"delete": { "confirm": "will delete the record", "input": { "schema": {} } }
			}
		}
	}`))
	if err != nil {
		t.Fatalf("ParseManifest returned error: %v", err)
	}
	command := manifest.Commands[0]
	if command.Confirm != "will delete the record" {
		t.Fatalf("expected string confirm preserved, got %q", command.Confirm)
	}
	if command.InputVia != "stdin" {
		t.Fatalf("expected default input via stdin, got %q", command.InputVia)
	}
	if !command.HasInputSchema {
		t.Fatal("expected empty schema object to count as schema")
	}
}

func TestManifestValidateReportsAllIssueKinds(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "real.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write real.sh: %v", err)
	}

	manifest, err := ParseManifest([]byte(`{
		"name": "other-name",
		"scripts": {
			"ok": "bash scripts/real.sh",
			"broken": "bash scripts/missing.sh"
		},
		"skm": {
			"runtime": { "setup": "ghost" },
			"commands": {
				"ghost-command": { "description": "no scripts entry" }
			}
		}
	}`))
	if err != nil {
		t.Fatalf("ParseManifest returned error: %v", err)
	}

	issues, codes := manifest.Validate(root, "My Skill")
	byCode := map[string]int{}
	for _, issue := range issues {
		byCode[issue.Code]++
	}
	if byCode["manifest_command_missing"] != 2 {
		t.Fatalf("expected 2 manifest_command_missing issues (leftover annotation + setup), got %d", byCode["manifest_command_missing"])
	}
	if byCode["manifest_target_missing"] != 1 {
		t.Fatalf("expected 1 manifest_target_missing issue, got %d", byCode["manifest_target_missing"])
	}
	if byCode["manifest_name_mismatch"] != 1 {
		t.Fatalf("expected 1 manifest_name_mismatch issue, got %d", byCode["manifest_name_mismatch"])
	}
	for _, code := range []string{"manifest_command_missing", "manifest_target_missing", "manifest_name_mismatch"} {
		if !containsString(codes, code) {
			t.Fatalf("expected code %s in %v", code, codes)
		}
	}

	var targetIssue *discoveredIssue
	for index := range issues {
		if issues[index].Code == "manifest_target_missing" {
			targetIssue = &issues[index]
		}
	}
	if targetIssue == nil || targetIssue.RelativePath != ManifestFileName {
		t.Fatalf("expected manifest_target_missing issue anchored to %s, got %+v", ManifestFileName, targetIssue)
	}
	if !strings.Contains(targetIssue.Message, "scripts/missing.sh") {
		t.Fatalf("expected missing target in message, got %q", targetIssue.Message)
	}
}

func TestManifestValidateNameMatchesBySlug(t *testing.T) {
	manifest, err := ParseManifest([]byte(`{"name": "browser-auth", "scripts": {"login": "sh login.sh"}}`))
	if err != nil {
		t.Fatalf("ParseManifest returned error: %v", err)
	}
	issues, codes := manifest.Validate(t.TempDir(), "Browser Auth")
	if len(issues) != 0 || len(codes) != 0 {
		t.Fatalf("expected slug-equal names to validate clean, got issues=%v codes=%v", issues, codes)
	}
}

func TestDiscoverProviderParsesManifestCommandsAndIssues(t *testing.T) {
	root := t.TempDir()

	validDir := filepath.Join(root, "with-manifest")
	if err := os.MkdirAll(filepath.Join(validDir, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir valid skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "SKILL.md"), []byte(`---
name: with-manifest
---

Body`), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "scripts", "run.sh"), []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatalf("write run.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validDir, ManifestFileName), []byte(`{
		"name": "with-manifest",
		"scripts": { "run": "bash scripts/run.sh" },
		"skm": { "commands": { "run": { "description": "Run it", "confirm": true } } }
	}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	invalidDir := filepath.Join(root, "bad-manifest")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("mkdir bad manifest skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(invalidDir, "SKILL.md"), []byte(`---
name: bad-manifest
---

Body`), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(invalidDir, ManifestFileName), []byte(`{ not json`), 0o644); err != nil {
		t.Fatalf("write broken manifest: %v", err)
	}

	skills, issues, err := discoverProvider(&models.Provider{RootPath: root, ScanMode: "shallow"})
	if err != nil {
		t.Fatalf("discoverProvider returned error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 discovered skills, got %d", len(skills))
	}

	var withManifest, badManifest *discoveredSkill
	for index := range skills {
		switch skills[index].DirectoryName {
		case "with-manifest":
			withManifest = &skills[index]
		case "bad-manifest":
			badManifest = &skills[index]
		}
	}
	if withManifest == nil || badManifest == nil {
		t.Fatalf("expected both fixture skills, got %v", skills)
	}

	if withManifest.Status != "ready" {
		t.Fatalf("manifest issues must not flip skill status, got %q", withManifest.Status)
	}
	if len(withManifest.Commands) != 1 {
		t.Fatalf("expected 1 command on with-manifest, got %d", len(withManifest.Commands))
	}
	command := withManifest.Commands[0]
	if command.Name != "run" || command.Line != "bash scripts/run.sh" || command.Confirm == "" {
		t.Fatalf("unexpected command %+v", command)
	}

	var sawInvalidManifest bool
	for _, issue := range issues {
		if issue.Code == "manifest_invalid_json" && issue.RootPath == invalidDir {
			sawInvalidManifest = true
			if issue.Severity != "error" {
				t.Fatalf("expected error severity, got %q", issue.Severity)
			}
		}
	}
	if !sawInvalidManifest {
		t.Fatal("expected manifest_invalid_json issue for bad-manifest")
	}
	if !containsString(badManifest.IssueCodes, "manifest_invalid_json") {
		t.Fatalf("expected manifest_invalid_json in issue codes, got %v", badManifest.IssueCodes)
	}
	if len(badManifest.Commands) != 0 {
		t.Fatalf("expected no commands for broken manifest, got %v", badManifest.Commands)
	}
}
