package service

import (
	"backend-go/internal/models"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverProviderFlagsMissingSkillAndNameMismatch(t *testing.T) {
	root := t.TempDir()

	validDir := filepath.Join(root, "valid-skill")
	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatalf("mkdir valid skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "SKILL.md"), []byte(`---
name: Valid Skill
category: utilities
tags: [catalog]
---

Body`), 0o644); err != nil {
		t.Fatalf("write valid SKILL.md: %v", err)
	}

	mismatchDir := filepath.Join(root, "wrong-folder")
	if err := os.MkdirAll(mismatchDir, 0o755); err != nil {
		t.Fatalf("mkdir mismatch skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mismatchDir, "SKILL.md"), []byte(`---
name: Different Name
---

Body`), 0o644); err != nil {
		t.Fatalf("write mismatch SKILL.md: %v", err)
	}

	missingDir := filepath.Join(root, "missing-skill-md")
	if err := os.MkdirAll(missingDir, 0o755); err != nil {
		t.Fatalf("mkdir missing skill dir: %v", err)
	}

	skills, issues, err := discoverProvider(&models.Provider{RootPath: root, ScanMode: "shallow"})
	if err != nil {
		t.Fatalf("discoverProvider returned error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 discovered skills, got %d", len(skills))
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	var sawMissingSkillMD bool
	var sawMismatch bool
	for _, issue := range issues {
		switch issue.Code {
		case "missing_skill_md":
			sawMissingSkillMD = true
		case "name_directory_mismatch":
			sawMismatch = true
		}
	}
	if !sawMissingSkillMD {
		t.Fatal("expected missing_skill_md issue")
	}
	if !sawMismatch {
		t.Fatal("expected name_directory_mismatch issue")
	}
}

func TestDiscoverProviderRecursiveFindsNestedSkills(t *testing.T) {
	root := t.TempDir()
	nestedSkillDir := filepath.Join(root, "catalog", "nested-skill")
	if err := os.MkdirAll(nestedSkillDir, 0o755); err != nil {
		t.Fatalf("mkdir nested skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedSkillDir, "SKILL.md"), []byte(`---
name: Nested Skill
category: catalog
---

Nested body`), 0o644); err != nil {
		t.Fatalf("write nested SKILL.md: %v", err)
	}

	skills, issues, err := discoverProvider(&models.Provider{RootPath: root, ScanMode: "recursive"})
	if err != nil {
		t.Fatalf("discoverProvider returned error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(issues))
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 discovered skill, got %d", len(skills))
	}
	if skills[0].DirectoryName != "nested-skill" {
		t.Fatalf("expected nested-skill directory, got %q", skills[0].DirectoryName)
	}
}
