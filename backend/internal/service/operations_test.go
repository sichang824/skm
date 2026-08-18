package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const operationsManifest = `{
	"name": "fixture-skill",
	"version": "1.0.0",
	"scripts": {
		"prep": "bash scripts/prep.sh",
		"transcribe": "bash scripts/transcribe.sh",
		"claims:delete": "node operations.js claims.delete"
	},
	"skm": {
		"schemaVersion": 1,
		"runtime": { "env": ["FIXTURE_KEY"], "setup": "prep" },
		"commands": {
			"transcribe": { "description": "Transcribe a local audio file" },
			"claims:delete": {
				"description": "Delete a claim",
				"confirm": "will delete the claim",
				"input": { "via": "argv", "schema": { "type": "object" } }
			}
		}
	}
}`

func newOperationsFixture(t *testing.T, skillMd string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ManifestFileName), []byte(operationsManifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return root
}

func readSkillMd(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	return string(data)
}

func TestRenderOperationsSection(t *testing.T) {
	manifest, err := ParseManifest([]byte(operationsManifest))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	section, count := RenderOperationsSection("SKIL0001", manifest)
	if count != 2 {
		t.Fatalf("expected 2 rendered commands (setup excluded), got %d", count)
	}
	for _, expected := range []string{
		"## Operations",
		operationsBeginMark,
		operationsEndMark,
		"### transcribe",
		"Transcribe a local audio file",
		"Command: `skm skills exec SKIL0001 transcribe -- <args...>`",
		"### claims:delete",
		"> ⚠️ Requires confirmation: will delete the claim",
		"Command: `skm skills exec SKIL0001 claims:delete --input '<json>'`",
		"`FIXTURE_KEY`",
	} {
		if !strings.Contains(section, expected) {
			t.Fatalf("section missing %q\n--- section ---\n%s", expected, section)
		}
	}
	if strings.Contains(section, "### prep") {
		t.Fatalf("setup command must not be rendered as an operation\n%s", section)
	}
	if strings.Contains(section, "<skill-zid>") {
		t.Fatalf("known zid must replace the placeholder\n%s", section)
	}

	placeholder, _ := RenderOperationsSection("", manifest)
	if !strings.Contains(placeholder, "skm skills exec <skill-zid> transcribe") {
		t.Fatalf("empty zid must keep the placeholder\n%s", placeholder)
	}
}

func TestGenerateOperationsAppendsSection(t *testing.T) {
	root := newOperationsFixture(t, "---\nname: fixture-skill\n---\n\n# Fixture\n\nBody text.\n")

	result, err := GenerateOperations(root, "SKIL0001", "fixture-skill", false)
	if err != nil {
		t.Fatalf("GenerateOperations: %v", err)
	}
	if !result.Changed || !result.Written || result.CommandCount != 2 {
		t.Fatalf("unexpected result %+v", result)
	}

	content := readSkillMd(t, root)
	if !strings.HasPrefix(content, "---\nname: fixture-skill\n---\n\n# Fixture\n\nBody text.") {
		t.Fatalf("existing content must be preserved:\n%s", content)
	}
	if !strings.Contains(content, "## Operations") || !strings.Contains(content, operationsEndMark) {
		t.Fatalf("section must be appended:\n%s", content)
	}
	if strings.Count(content, "## Operations") != 1 {
		t.Fatalf("expected exactly one Operations heading:\n%s", content)
	}
}

func TestGenerateOperationsIsIdempotent(t *testing.T) {
	root := newOperationsFixture(t, "# Fixture\n\nBody text.\n")

	if _, err := GenerateOperations(root, "SKIL0001", "fixture-skill", false); err != nil {
		t.Fatalf("first generation: %v", err)
	}
	afterFirst := readSkillMd(t, root)

	result, err := GenerateOperations(root, "SKIL0001", "fixture-skill", false)
	if err != nil {
		t.Fatalf("second generation: %v", err)
	}
	if result.Changed || result.Written {
		t.Fatalf("second generation must be a no-op, got %+v", result)
	}
	if readSkillMd(t, root) != afterFirst {
		t.Fatalf("idempotent generation must not modify the file")
	}
}

func TestGenerateOperationsReplacesMarkedBlock(t *testing.T) {
	root := newOperationsFixture(t, "# Fixture\n\nBody text.\n")
	if _, err := GenerateOperations(root, "SKIL0001", "fixture-skill", false); err != nil {
		t.Fatalf("seed generation: %v", err)
	}

	// Change the manifest, then regenerate: the marked block is replaced.
	manifest := strings.Replace(operationsManifest, "Transcribe a local audio file", "Transcribe audio fast", 1)
	if err := os.WriteFile(filepath.Join(root, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatalf("update manifest: %v", err)
	}
	result, err := GenerateOperations(root, "SKIL0001", "fixture-skill", false)
	if err != nil {
		t.Fatalf("regeneration: %v", err)
	}
	if !result.Written {
		t.Fatalf("expected rewrite after manifest change, got %+v", result)
	}

	content := readSkillMd(t, root)
	if !strings.Contains(content, "Transcribe audio fast") {
		t.Fatalf("regenerated section must reflect the manifest:\n%s", content)
	}
	if strings.Contains(content, "Transcribe a local audio file") {
		t.Fatalf("stale generated text must be gone:\n%s", content)
	}
	if strings.Count(content, operationsBeginMark) != 1 || strings.Count(content, "## Operations") != 1 {
		t.Fatalf("expected exactly one generated block:\n%s", content)
	}
	if !strings.Contains(content, "# Fixture") || !strings.Contains(content, "Body text.") {
		t.Fatalf("surrounding content must survive:\n%s", content)
	}
}

func TestGenerateOperationsReplacesUnmarkedShijuSection(t *testing.T) {
	shijuStyle := "# Fixture\n\nIntro.\n\n## Operations\n\n\n### old.command\n\nOld prose\n\nCommand: `npm run old:command -- '<input-json>'`\n\n## Next Steps\n\nKeep me.\n"
	root := newOperationsFixture(t, shijuStyle)

	result, err := GenerateOperations(root, "SKIL0001", "fixture-skill", false)
	if err != nil {
		t.Fatalf("GenerateOperations: %v", err)
	}
	if !result.Written {
		t.Fatalf("expected replacement of unmarked section, got %+v", result)
	}

	content := readSkillMd(t, root)
	if strings.Contains(content, "old.command") || strings.Contains(content, "npm run") {
		t.Fatalf("legacy section content must be replaced:\n%s", content)
	}
	if !strings.Contains(content, "## Next Steps") || !strings.Contains(content, "Keep me.") {
		t.Fatalf("sections after Operations must survive:\n%s", content)
	}
	if !strings.Contains(content, "### transcribe") {
		t.Fatalf("generated commands must be present:\n%s", content)
	}
}

func TestGenerateOperationsCheckMode(t *testing.T) {
	root := newOperationsFixture(t, "# Fixture\n")

	result, err := GenerateOperations(root, "SKIL0001", "fixture-skill", true)
	if err != nil {
		t.Fatalf("check mode: %v", err)
	}
	if !result.Changed || result.Written {
		t.Fatalf("check mode must report change without writing, got %+v", result)
	}
	if strings.Contains(readSkillMd(t, root), "## Operations") {
		t.Fatalf("check mode must not touch the file")
	}

	if _, err := GenerateOperations(root, "SKIL0001", "fixture-skill", false); err != nil {
		t.Fatalf("real generation: %v", err)
	}
	stable, err := GenerateOperations(root, "SKIL0001", "fixture-skill", true)
	if err != nil {
		t.Fatalf("second check: %v", err)
	}
	if stable.Changed {
		t.Fatalf("check after generation must be stable, got %+v", stable)
	}
}

func TestGenerateOperationsRequiresManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if _, err := GenerateOperations(root, "SKIL0001", "x", false); err == nil {
		t.Fatalf("expected manifest error")
	}
}
