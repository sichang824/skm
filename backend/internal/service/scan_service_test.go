package service

import (
	"backend-go/internal/models"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestPersistScanDedupesDuplicateDiscoveredRoots(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewScanService(db)
	provider := createTestProvider(t, db, "Agents Global", t.TempDir())
	providerID := provider.ID
	job := &models.ScanJob{
		ProviderID: &providerID,
		Scope:      "provider",
		StartedAt:  time.Now(),
		Status:     "running",
		LogLines:   []string{},
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("create scan job: %v", err)
	}

	rootPath := filepath.Join(provider.RootPath, "tdd-workflow")
	skillMdPath := filepath.Join(rootPath, "SKILL.md")
	modifiedAt := time.Now()
	discovered := discoveredSkill{
		RootPath:       rootPath,
		SkillMdPath:    skillMdPath,
		DirectoryName:  "tdd-workflow",
		Name:           "tdd-workflow",
		Slug:           "tdd-workflow",
		Status:         "ready",
		ContentHash:    "hash-1",
		LastModifiedAt: &modifiedAt,
		IssueCodes:     []string{},
		Tags:           []string{},
	}

	_, addedCount, removedCount, changedCount, invalidCount, err := service.persistScan(context.Background(), provider, job, []discoveredSkill{discovered, discovered}, nil)
	if err != nil {
		t.Fatalf("persistScan returned error: %v", err)
	}
	if addedCount != 1 {
		t.Fatalf("expected addedCount=1, got %d", addedCount)
	}
	if removedCount != 0 || changedCount != 0 || invalidCount != 0 {
		t.Fatalf("unexpected counters removed=%d changed=%d invalid=%d", removedCount, changedCount, invalidCount)
	}

	var skills []models.Skill
	if err := db.Where("provider_id = ?", provider.ID).Find(&skills).Error; err != nil {
		t.Fatalf("load saved skills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 saved skill, got %d", len(skills))
	}
	if skills[0].RootPath != rootPath {
		t.Fatalf("unexpected root path: got %q want %q", skills[0].RootPath, rootPath)
	}
	if skills[0].SkillMdPath != skillMdPath {
		t.Fatalf("unexpected skill md path: got %q want %q", skills[0].SkillMdPath, skillMdPath)
	}
	if skills[0].LastScannedAt.IsZero() {
		t.Fatal("expected LastScannedAt to be set")
	}
	if skills[0].ProviderID != provider.ID {
		t.Fatalf("unexpected provider id: got %d want %d", skills[0].ProviderID, provider.ID)
	}
	if _, _, _, _, _, err := service.persistScan(context.Background(), provider, job, []discoveredSkill{discovered}, nil); err != nil {
		t.Fatalf("persistScan should update existing record without error: %v", err)
	}
	var total int64
	if err := db.Model(&models.Skill{}).Where("provider_id = ?", provider.ID).Count(&total).Error; err != nil {
		t.Fatalf("count saved skills: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total skills=1 after re-persist, got %d", total)
	}
}

func TestPersistScanRevivesSoftDeletedSkill(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewScanService(db)
	provider := createTestProvider(t, db, "Agents Global", t.TempDir())
	providerID := provider.ID
	job := &models.ScanJob{
		ProviderID: &providerID,
		Scope:      "provider",
		StartedAt:  time.Now(),
		Status:     "running",
		LogLines:   []string{},
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("create scan job: %v", err)
	}

	rootPath := filepath.Join(provider.RootPath, "skill-jira")
	skillMdPath := filepath.Join(rootPath, "SKILL.md")
	modifiedAt := time.Now()
	existing := &models.Skill{
		ProviderID:     provider.ID,
		Name:           "skill-jira",
		Slug:           "skill-jira",
		DirectoryName:  "skill-jira",
		RootPath:       rootPath,
		SkillMdPath:    skillMdPath,
		Status:         "ready",
		Tags:           []string{},
		IssueCodes:     []string{},
		ConflictKinds:  []string{},
		LastScannedAt:  modifiedAt,
		LastModifiedAt: &modifiedAt,
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("create existing skill: %v", err)
	}
	if err := db.Delete(existing).Error; err != nil {
		t.Fatalf("soft delete existing skill: %v", err)
	}

	discovered := discoveredSkill{
		RootPath:       rootPath,
		SkillMdPath:    skillMdPath,
		DirectoryName:  "skill-jira",
		Name:           "skill-jira",
		Slug:           "skill-jira",
		Status:         "ready",
		ContentHash:    "hash-1",
		LastModifiedAt: &modifiedAt,
		IssueCodes:     []string{},
		Tags:           []string{},
	}

	_, addedCount, removedCount, changedCount, invalidCount, err := service.persistScan(context.Background(), provider, job, []discoveredSkill{discovered}, nil)
	if err != nil {
		t.Fatalf("persistScan returned error: %v", err)
	}
	if addedCount != 1 || removedCount != 0 || changedCount != 0 || invalidCount != 0 {
		t.Fatalf("unexpected counters added=%d removed=%d changed=%d invalid=%d", addedCount, removedCount, changedCount, invalidCount)
	}

	var skills []models.Skill
	if err := db.Unscoped().Where("provider_id = ?", provider.ID).Find(&skills).Error; err != nil {
		t.Fatalf("load skills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill record after revive, got %d", len(skills))
	}
	if skills[0].DeletedAt.Valid {
		t.Fatal("expected revived skill to be active, got soft-deleted row")
	}
	if skills[0].ID != existing.ID {
		t.Fatalf("expected revive to reuse existing row id %d, got %d", existing.ID, skills[0].ID)
	}
}
