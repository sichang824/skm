package service

import (
	"backend-go/internal/models"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildConflictGroupsClassifiesContentDiff(t *testing.T) {
	now := time.Now()
	skills := []models.Skill{
		{
			BaseModel:     models.BaseModel{Zid: "SKIL0001"},
			Name:          "Shared Skill",
			RootPath:      "/tmp/provider-a/shared-skill",
			ContentHash:   "hash-a",
			Provider:      models.Provider{Priority: 200, Enabled: true},
			LastScannedAt: now,
		},
		{
			BaseModel:     models.BaseModel{Zid: "SKIL0002"},
			Name:          "Shared Skill",
			RootPath:      "/tmp/provider-b/shared-skill",
			ContentHash:   "hash-b",
			Provider:      models.Provider{Priority: 100, Enabled: true},
			LastScannedAt: now.Add(-time.Minute),
		},
	}

	groups := buildConflictGroups(skills)
	if len(groups) != 1 {
		t.Fatalf("expected 1 conflict group, got %d", len(groups))
	}
	if groups[0].Kind != "name_content_diff" {
		t.Fatalf("expected name_content_diff, got %q", groups[0].Kind)
	}
	if groups[0].EffectiveSkillZid != "SKIL0001" {
		t.Fatalf("expected higher priority skill to win, got %q", groups[0].EffectiveSkillZid)
	}
}

func TestBuildConflictGroupsIgnoresAttachedSkills(t *testing.T) {
	now := time.Now()
	skills := []models.Skill{
		{
			BaseModel:     models.BaseModel{Zid: "SKIL0001"},
			Name:          "Shared Skill",
			RootPath:      "/tmp/provider-a/shared-skill",
			ContentHash:   "hash-a",
			Provider:      models.Provider{Priority: 200, Enabled: true},
			LastScannedAt: now,
		},
		{
			BaseModel:     models.BaseModel{Zid: "SKIL0002"},
			Name:          "Shared Skill",
			RootPath:      "/tmp/provider-b/shared-skill-copy",
			ContentHash:   "hash-a",
			Provider:      models.Provider{Priority: 100, Enabled: true},
			LastScannedAt: now.Add(-time.Minute),
			Relation:      &models.SkillRelation{Mode: "from", FromPath: "/tmp/provider-a/shared-skill"},
		},
	}

	groups := buildConflictGroups(skills)
	if len(groups) != 0 {
		t.Fatalf("expected attached skills to be ignored in conflict groups, got %d groups", len(groups))
	}
}

func TestRebuildConflictStateRepairsInvalidConflictKindsJSON(t *testing.T) {
	db := openCatalogTestDB(t)
	if err := db.AutoMigrate(models.ModelsForAutoMigrate...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	providerA := models.Provider{Name: "Provider A", Type: "local", RootPath: "/tmp/provider-a", Enabled: true, Priority: 200, ScanMode: "recursive", LastScanStatus: "never"}
	providerB := models.Provider{Name: "Provider B", Type: "local", RootPath: "/tmp/provider-b", Enabled: true, Priority: 100, ScanMode: "recursive", LastScanStatus: "never"}
	if err := db.Create(&providerA).Error; err != nil {
		t.Fatalf("create provider A: %v", err)
	}
	if err := db.Create(&providerB).Error; err != nil {
		t.Fatalf("create provider B: %v", err)
	}

	now := time.Now()
	skillA := models.Skill{
		ProviderID:    providerA.ID,
		Name:          "Shared Skill",
		Slug:          "shared-skill",
		DirectoryName: "shared-skill",
		RootPath:      "/tmp/provider-a/shared-skill",
		Status:        "ready",
		LastScannedAt: now,
		ContentHash:   "hash-a",
		Tags:          []string{},
		IssueCodes:    []string{},
		ConflictKinds: []string{},
		IsEffective:   true,
	}
	skillB := models.Skill{
		ProviderID:    providerB.ID,
		Name:          "Shared Skill",
		Slug:          "shared-skill",
		DirectoryName: "shared-skill",
		RootPath:      "/tmp/provider-b/shared-skill",
		Status:        "ready",
		LastScannedAt: now.Add(-time.Minute),
		ContentHash:   "hash-b",
		Tags:          []string{},
		IssueCodes:    []string{},
		ConflictKinds: []string{},
		IsEffective:   true,
	}
	if err := db.Create(&skillA).Error; err != nil {
		t.Fatalf("create skill A: %v", err)
	}
	if err := db.Create(&skillB).Error; err != nil {
		t.Fatalf("create skill B: %v", err)
	}

	if err := db.Exec("UPDATE skills SET conflict_kinds = 'name_duplicate' WHERE id IN (?, ?)", skillA.ID, skillB.ID).Error; err != nil {
		t.Fatalf("corrupt conflict_kinds: %v", err)
	}

	count, err := rebuildConflictState(db)
	if err != nil {
		t.Fatalf("rebuild conflict state: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 conflict group, got %d", count)
	}

	var invalidCount int64
	if err := db.Raw("SELECT COUNT(*) FROM skills WHERE conflict_kinds IS NOT NULL AND json_valid(conflict_kinds) = 0").Scan(&invalidCount).Error; err != nil {
		t.Fatalf("count invalid conflict_kinds: %v", err)
	}
	if invalidCount != 0 {
		t.Fatalf("expected conflict_kinds to be repaired, got %d invalid rows", invalidCount)
	}
}

func TestRebuildConflictStateIgnoresAttachedSkills(t *testing.T) {
	db := openCatalogTestDB(t)

	baseDir := t.TempDir()
	rootA := filepath.Join(baseDir, "provider-a", "shared-skill")
	rootB := filepath.Join(baseDir, "provider-b", "shared-skill-copy")
	for _, root := range []string{rootA, rootB} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("mkdir root %s: %v", root, err)
		}
	}
	if err := os.WriteFile(filepath.Join(rootB, skillRelationFromFile), []byte(rootA+"\n"), 0o644); err != nil {
		t.Fatalf("write .from: %v", err)
	}

	providerA := models.Provider{Name: "Provider A", Type: "local", RootPath: filepath.Join(baseDir, "provider-a"), Enabled: true, Priority: 200, ScanMode: "recursive", LastScanStatus: "never"}
	providerB := models.Provider{Name: "Provider B", Type: "local", RootPath: filepath.Join(baseDir, "provider-b"), Enabled: true, Priority: 100, ScanMode: "recursive", LastScanStatus: "never"}
	if err := db.Create(&providerA).Error; err != nil {
		t.Fatalf("create provider A: %v", err)
	}
	if err := db.Create(&providerB).Error; err != nil {
		t.Fatalf("create provider B: %v", err)
	}

	now := time.Now()
	skillA := models.Skill{
		ProviderID:    providerA.ID,
		Name:          "Shared Skill",
		Slug:          "shared-skill",
		DirectoryName: "shared-skill",
		RootPath:      rootA,
		Status:        "ready",
		LastScannedAt: now,
		ContentHash:   "hash-a",
		Tags:          []string{},
		IssueCodes:    []string{},
		ConflictKinds: []string{},
		IsEffective:   true,
	}
	skillB := models.Skill{
		ProviderID:    providerB.ID,
		Name:          "Shared Skill",
		Slug:          "shared-skill",
		DirectoryName: "shared-skill-copy",
		RootPath:      rootB,
		Status:        "ready",
		LastScannedAt: now.Add(-time.Minute),
		ContentHash:   "hash-a",
		Tags:          []string{},
		IssueCodes:    []string{},
		ConflictKinds: []string{},
		IsEffective:   true,
	}
	if err := db.Create(&skillA).Error; err != nil {
		t.Fatalf("create skill A: %v", err)
	}
	if err := db.Create(&skillB).Error; err != nil {
		t.Fatalf("create skill B: %v", err)
	}

	count, err := rebuildConflictState(db)
	if err != nil {
		t.Fatalf("rebuild conflict state: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 conflict groups when one skill is attached, got %d", count)
	}

	var refreshedA models.Skill
	var refreshedB models.Skill
	if err := db.First(&refreshedA, skillA.ID).Error; err != nil {
		t.Fatalf("reload skill A: %v", err)
	}
	if err := db.First(&refreshedB, skillB.ID).Error; err != nil {
		t.Fatalf("reload skill B: %v", err)
	}
	if refreshedA.IsConflict || refreshedB.IsConflict {
		t.Fatalf("expected attached skills to be excluded from conflicts, got A=%v B=%v", refreshedA.IsConflict, refreshedB.IsConflict)
	}
	if len(refreshedA.ConflictKinds) != 0 || len(refreshedB.ConflictKinds) != 0 {
		t.Fatalf("expected empty conflict kinds, got A=%#v B=%#v", refreshedA.ConflictKinds, refreshedB.ConflictKinds)
	}
}
