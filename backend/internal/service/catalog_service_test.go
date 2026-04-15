package service

import (
	"backend-go/internal/models"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAttachSkillMoveMovesDirectory(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	sourceRoot := filepath.Join(baseDir, "source")
	targetRoot := filepath.Join(baseDir, "target")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source root: %v", err)
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		t.Fatalf("mkdir target root: %v", err)
	}

	sourceProvider := createTestProvider(t, db, "Source", sourceRoot)
	targetProvider := createTestProvider(t, db, "Target", targetRoot)
	skill := createTestSkill(t, db, sourceProvider, filepath.Join(sourceRoot, "alpha-skill"), "alpha_skill")

	result, err := service.AttachSkill(ctx, skill.Zid, SkillAttachInput{TargetProviderZid: targetProvider.Zid, Mode: "move"})
	if err != nil {
		t.Fatalf("AttachSkill move returned error: %v", err)
	}
	if _, err := os.Stat(skill.RootPath); !os.IsNotExist(err) {
		t.Fatalf("expected source path to be moved, stat err=%v", err)
	}
	if _, err := os.Stat(result.TargetPath); err != nil {
		t.Fatalf("expected target path to exist: %v", err)
	}
	if got, want := result.TargetPath, filepath.Join(targetRoot, skill.DirectoryName); got != want {
		t.Fatalf("unexpected target path: got %s want %s", got, want)
	}
}

func TestAttachSkillAttachCopiesFilesAndWritesRelationMetadata(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	sourceRoot := filepath.Join(baseDir, "source")
	targetRoot := filepath.Join(baseDir, "target")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source root: %v", err)
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		t.Fatalf("mkdir target root: %v", err)
	}

	sourceProvider := createTestProvider(t, db, "Source Link", sourceRoot)
	targetProvider := createTestProvider(t, db, "Target Link", targetRoot)
	skill := createTestSkill(t, db, sourceProvider, filepath.Join(sourceRoot, "linked-skill"), "linked_skill")
	if err := os.WriteFile(filepath.Join(skill.RootPath, "notes.md"), []byte("source copy"), 0o644); err != nil {
		t.Fatalf("write notes.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(targetRoot, skill.DirectoryName), 0o755); err != nil {
		t.Fatalf("mkdir target skill root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetRoot, skill.DirectoryName, "notes.md"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale notes.md: %v", err)
	}

	result, err := service.AttachSkill(ctx, skill.Zid, SkillAttachInput{TargetProviderZid: targetProvider.Zid, Mode: "attach"})
	if err != nil {
		t.Fatalf("AttachSkill attach returned error: %v", err)
	}
	info, err := os.Lstat(result.TargetPath)
	if err != nil {
		t.Fatalf("lstat target path: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected target path to be a directory")
	}
	content, err := os.ReadFile(filepath.Join(result.TargetPath, "notes.md"))
	if err != nil {
		t.Fatalf("read copied notes.md: %v", err)
	}
	if string(content) != "source copy" {
		t.Fatalf("expected copied file content to be overwritten, got %q", string(content))
	}
	fromContent, err := os.ReadFile(filepath.Join(result.TargetPath, skillRelationFromFile))
	if err != nil {
		t.Fatalf("read .from: %v", err)
	}
	if got := strings.TrimSpace(string(fromContent)); got != skill.RootPath {
		t.Fatalf("unexpected .from content: got %q want %q", got, skill.RootPath)
	}
	toContent, err := os.ReadFile(filepath.Join(skill.RootPath, skillRelationToFile))
	if err != nil {
		t.Fatalf("read .to: %v", err)
	}
	var metadata skillToMetadata
	if err := json.Unmarshal(toContent, &metadata); err != nil {
		t.Fatalf("unmarshal .to: %v", err)
	}
	if len(metadata.Directories) != 1 || metadata.Directories[0] != result.TargetPath {
		t.Fatalf("unexpected .to directories: %#v", metadata.Directories)
	}
	if len(metadata.Files) != 2 || metadata.Files[0] != "SKILL.md" || metadata.Files[1] != "notes.md" {
		t.Fatalf("unexpected .to files: %#v", metadata.Files)
	}
}

func TestAttachSkillAttachAppendsTargetDirectories(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	sourceRoot := filepath.Join(baseDir, "source")
	targetRootA := filepath.Join(baseDir, "target-a")
	targetRootB := filepath.Join(baseDir, "target-b")
	for _, root := range []string{sourceRoot, targetRootA, targetRootB} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("mkdir root %s: %v", root, err)
		}
	}

	sourceProvider := createTestProvider(t, db, "Source Attach", sourceRoot)
	targetProviderA := createTestProvider(t, db, "Target Attach A", targetRootA)
	targetProviderB := createTestProvider(t, db, "Target Attach B", targetRootB)
	skill := createTestSkill(t, db, sourceProvider, filepath.Join(sourceRoot, "copied-skill"), "copied_skill")

	if _, err := service.AttachSkill(ctx, skill.Zid, SkillAttachInput{TargetProviderZid: targetProviderA.Zid, Mode: "attach"}); err != nil {
		t.Fatalf("first attach returned error: %v", err)
	}
	secondResult, err := service.AttachSkill(ctx, skill.Zid, SkillAttachInput{TargetProviderZid: targetProviderB.Zid, Mode: "attach"})
	if err != nil {
		t.Fatalf("second attach returned error: %v", err)
	}

	toContent, err := os.ReadFile(filepath.Join(skill.RootPath, skillRelationToFile))
	if err != nil {
		t.Fatalf("read .to: %v", err)
	}
	var metadata skillToMetadata
	if err := json.Unmarshal(toContent, &metadata); err != nil {
		t.Fatalf("unmarshal .to: %v", err)
	}
	if len(metadata.Directories) != 2 {
		t.Fatalf("expected 2 target directories, got %#v", metadata.Directories)
	}
	foundSecondTarget := false
	for _, directory := range metadata.Directories {
		if directory == secondResult.TargetPath {
			foundSecondTarget = true
			break
		}
	}
	if !foundSecondTarget {
		t.Fatalf("expected second target path %q in %#v", secondResult.TargetPath, metadata.Directories)
	}
	fromContent, err := os.ReadFile(filepath.Join(secondResult.TargetPath, skillRelationFromFile))
	if err != nil {
		t.Fatalf("read target .from: %v", err)
	}
	if strings.TrimSpace(string(fromContent)) != skill.RootPath {
		t.Fatalf("unexpected target .from content: %q", string(fromContent))
	}
}

func TestDeleteSkillRemovesDirectory(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	providerRoot := filepath.Join(baseDir, "provider")
	if err := os.MkdirAll(providerRoot, 0o755); err != nil {
		t.Fatalf("mkdir provider root: %v", err)
	}

	provider := createTestProvider(t, db, "Delete Provider", providerRoot)
	skill := createTestSkill(t, db, provider, filepath.Join(providerRoot, "delete-me"), "delete_me")

	result, err := service.DeleteSkill(ctx, skill.Zid)
	if err != nil {
		t.Fatalf("DeleteSkill returned error: %v", err)
	}
	if !result.Deleted {
		t.Fatal("expected deleted result to be true")
	}
	if _, err := os.Stat(skill.RootPath); !os.IsNotExist(err) {
		t.Fatalf("expected skill directory to be removed, stat err=%v", err)
	}
}

func TestSyncSkillRefreshesAttachedCopy(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	sourceRoot := filepath.Join(baseDir, "source")
	targetRoot := filepath.Join(baseDir, "target")
	for _, root := range []string{sourceRoot, targetRoot} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("mkdir root %s: %v", root, err)
		}
	}

	sourceProvider := createTestProvider(t, db, "Source Sync", sourceRoot)
	targetProvider := createTestProvider(t, db, "Target Sync", targetRoot)
	sourceSkill := createTestSkill(t, db, sourceProvider, filepath.Join(sourceRoot, "sync-skill"), "sync_skill")
	targetSkill := createTestSkill(t, db, targetProvider, filepath.Join(targetRoot, "sync-skill"), "sync_skill")

	if err := os.WriteFile(filepath.Join(sourceSkill.RootPath, "guide.md"), []byte("fresh content"), 0o644); err != nil {
		t.Fatalf("write source guide.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetSkill.RootPath, "guide.md"), []byte("stale content"), 0o644); err != nil {
		t.Fatalf("write target guide.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetSkill.RootPath, "obsolete.md"), []byte("obsolete"), 0o644); err != nil {
		t.Fatalf("write target obsolete.md: %v", err)
	}
	if err := writeSkillToMetadata(sourceSkill.RootPath, skillToMetadata{Files: []string{"SKILL.md", "guide.md"}, Directories: []string{targetSkill.RootPath}}); err != nil {
		t.Fatalf("write source .to: %v", err)
	}
	if err := writeSkillFromMetadata(targetSkill.RootPath, sourceSkill.RootPath); err != nil {
		t.Fatalf("write target .from: %v", err)
	}

	result, err := service.SyncSkill(ctx, targetSkill.Zid)
	if err != nil {
		t.Fatalf("SyncSkill returned error: %v", err)
	}
	if !result.Synced {
		t.Fatal("expected synced result to be true")
	}
	if result.SourcePath != sourceSkill.RootPath || result.TargetPath != targetSkill.RootPath {
		t.Fatalf("unexpected sync paths: %#v", result)
	}
	guideContent, err := os.ReadFile(filepath.Join(targetSkill.RootPath, "guide.md"))
	if err != nil {
		t.Fatalf("read synced guide.md: %v", err)
	}
	if string(guideContent) != "fresh content" {
		t.Fatalf("expected synced guide.md content, got %q", string(guideContent))
	}
	if _, err := os.Stat(filepath.Join(targetSkill.RootPath, "obsolete.md")); !os.IsNotExist(err) {
		t.Fatalf("expected obsolete.md to be removed, stat err=%v", err)
	}
	fromContent, err := os.ReadFile(filepath.Join(targetSkill.RootPath, skillRelationFromFile))
	if err != nil {
		t.Fatalf("read target .from: %v", err)
	}
	if strings.TrimSpace(string(fromContent)) != sourceSkill.RootPath {
		t.Fatalf("unexpected .from content: %q", string(fromContent))
	}
	toContent, err := os.ReadFile(filepath.Join(sourceSkill.RootPath, skillRelationToFile))
	if err != nil {
		t.Fatalf("read source .to: %v", err)
	}
	var metadata skillToMetadata
	if err := json.Unmarshal(toContent, &metadata); err != nil {
		t.Fatalf("unmarshal source .to: %v", err)
	}
	if len(metadata.Files) != 2 || metadata.Files[0] != "SKILL.md" || metadata.Files[1] != "guide.md" {
		t.Fatalf("unexpected synced files metadata: %#v", metadata.Files)
	}
	if len(metadata.Directories) != 1 || metadata.Directories[0] != targetSkill.RootPath {
		t.Fatalf("unexpected synced directories metadata: %#v", metadata.Directories)
	}
}

func TestSyncSkillRejectsNonAttachedSkill(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	providerRoot := filepath.Join(t.TempDir(), "provider")
	if err := os.MkdirAll(providerRoot, 0o755); err != nil {
		t.Fatalf("mkdir provider root: %v", err)
	}
	provider := createTestProvider(t, db, "Sync Reject", providerRoot)
	skill := createTestSkill(t, db, provider, filepath.Join(providerRoot, "plain-skill"), "plain_skill")

	if _, err := service.SyncSkill(ctx, skill.Zid); err == nil {
		t.Fatal("expected SyncSkill to reject non-attached skill")
	}
}

func TestListSkillsGroupedByRelation(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	sourceRoot := filepath.Join(baseDir, "source")
	targetRoot := filepath.Join(baseDir, "target")
	extraRoot := filepath.Join(baseDir, "extra")
	for _, root := range []string{sourceRoot, targetRoot, extraRoot} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("mkdir root %s: %v", root, err)
		}
	}

	sourceProvider := createTestProvider(t, db, "Source Group", sourceRoot)
	targetProvider := createTestProvider(t, db, "Target Group", targetRoot)
	extraProvider := createTestProvider(t, db, "Extra Group", extraRoot)

	toSkill := createTestSkill(t, db, sourceProvider, filepath.Join(sourceRoot, "grouped-skill"), "grouped_skill")
	fromSkill := createTestSkill(t, db, targetProvider, filepath.Join(targetRoot, "grouped-skill"), "grouped_skill_copy")
	plainSkill := createTestSkill(t, db, extraProvider, filepath.Join(extraRoot, "plain-skill"), "plain_skill")

	if err := writeSkillToMetadata(toSkill.RootPath, skillToMetadata{Files: []string{"SKILL.md"}, Directories: []string{fromSkill.RootPath}}); err != nil {
		t.Fatalf("write source .to: %v", err)
	}
	if err := writeSkillFromMetadata(fromSkill.RootPath, toSkill.RootPath); err != nil {
		t.Fatalf("write target .from: %v", err)
	}

	skills, err := service.ListSkills(ctx, SkillListFilters{Sort: "name", Grouped: true})
	if err != nil {
		t.Fatalf("ListSkills grouped returned error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 top-level skills, got %d", len(skills))
	}

	var groupedSkill *models.Skill
	for index := range skills {
		if skills[index].Zid == toSkill.Zid {
			groupedSkill = &skills[index]
		}
		if skills[index].Zid == fromSkill.Zid {
			t.Fatal("expected attached from skill to be nested instead of top-level")
		}
	}
	if groupedSkill == nil {
		t.Fatal("expected source to skill to remain in top-level list")
	}
	if groupedSkill.Relation == nil || groupedSkill.Relation.Mode != "to" {
		t.Fatalf("expected grouped source skill to keep to relation, got %#v", groupedSkill.Relation)
	}
	if len(groupedSkill.RelatedSkills) != 1 {
		t.Fatalf("expected 1 related skill, got %d", len(groupedSkill.RelatedSkills))
	}
	if groupedSkill.RelatedSkills[0].Zid != fromSkill.Zid {
		t.Fatalf("unexpected nested skill zid: got %s want %s", groupedSkill.RelatedSkills[0].Zid, fromSkill.Zid)
	}
	if groupedSkill.RelatedSkills[0].Relation == nil || groupedSkill.RelatedSkills[0].Relation.Mode != "from" {
		t.Fatalf("expected nested skill to keep from relation, got %#v", groupedSkill.RelatedSkills[0].Relation)
	}
	if skills[0].Zid != plainSkill.Zid && skills[1].Zid != plainSkill.Zid {
		t.Fatal("expected unrelated skill to remain top-level")
	}
}

func openCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "catalog-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Provider{}, &models.Skill{}, &models.ScanJob{}, &models.ScanIssue{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func createTestProvider(t *testing.T, db *gorm.DB, name, rootPath string) *models.Provider {
	t.Helper()
	provider := &models.Provider{Name: name, Type: "workspace", RootPath: rootPath, Enabled: true, Priority: 100, ScanMode: "recursive"}
	if err := db.Create(provider).Error; err != nil {
		t.Fatalf("create provider %s: %v", name, err)
	}
	return provider
}

func createTestSkill(t *testing.T, db *gorm.DB, provider *models.Provider, rootPath, skillName string) *models.Skill {
	t.Helper()
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("mkdir skill root: %v", err)
	}
	skillMdPath := filepath.Join(rootPath, "SKILL.md")
	content := "---\nname: " + skillName + "\n---\nsummary"
	if err := os.WriteFile(skillMdPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	now := time.Now()
	skill := &models.Skill{
		ProviderID:     provider.ID,
		Name:           skillName,
		Slug:           skillName,
		DirectoryName:  filepath.Base(rootPath),
		RootPath:       rootPath,
		SkillMdPath:    skillMdPath,
		Status:         "ready",
		Tags:           []string{},
		IssueCodes:     []string{},
		ConflictKinds:  []string{},
		LastScannedAt:  now,
		LastModifiedAt: &now,
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	return skill
}

func TestListIssuesLatestDedupesAcrossLatestJobs(t *testing.T) {
	db := openCatalogTestDB(t)

	providerA := models.Provider{Name: "Provider A", Type: "workspace", RootPath: "/tmp/provider-a", Enabled: true, Priority: 100, ScanMode: "recursive"}
	providerB := models.Provider{Name: "Provider B", Type: "workspace", RootPath: "/tmp/provider-b", Enabled: true, Priority: 90, ScanMode: "recursive"}
	if err := db.Create(&providerA).Error; err != nil {
		t.Fatalf("create provider A: %v", err)
	}
	if err := db.Create(&providerB).Error; err != nil {
		t.Fatalf("create provider B: %v", err)
	}

	olderJob := models.ScanJob{ProviderID: &providerA.ID, Scope: "provider", StartedAt: time.Now().Add(-2 * time.Hour), Status: "completed"}
	latestJobA := models.ScanJob{ProviderID: &providerA.ID, Scope: "provider", StartedAt: time.Now().Add(-time.Hour), Status: "completed"}
	latestJobB := models.ScanJob{ProviderID: &providerB.ID, Scope: "provider", StartedAt: time.Now().Add(-30 * time.Minute), Status: "completed"}
	if err := db.Create(&olderJob).Error; err != nil {
		t.Fatalf("create older job: %v", err)
	}
	if err := db.Create(&latestJobA).Error; err != nil {
		t.Fatalf("create latest job A: %v", err)
	}
	if err := db.Create(&latestJobB).Error; err != nil {
		t.Fatalf("create latest job B: %v", err)
	}

	staleIssue := models.ScanIssue{ScanJobID: olderJob.ID, ProviderID: &providerA.ID, RootPath: "/tmp/provider-a/skill-a", Code: "missing_name", Severity: "error", Message: "stale issue"}
	duplicateIssueA := models.ScanIssue{ScanJobID: latestJobA.ID, ProviderID: &providerA.ID, RootPath: "/tmp/provider-a/skill-a", Code: "missing_name", Severity: "error", Message: "current issue"}
	duplicateIssueB := models.ScanIssue{ScanJobID: latestJobA.ID, ProviderID: &providerA.ID, RootPath: "/tmp/provider-a/skill-a", Code: "missing_name", Severity: "error", Message: "current issue"}
	currentIssueB := models.ScanIssue{ScanJobID: latestJobB.ID, ProviderID: &providerB.ID, RootPath: "/tmp/provider-b/skill-b", Code: "frontmatter_parse_failed", Severity: "error", Message: "provider b issue"}
	if err := db.Create(&staleIssue).Error; err != nil {
		t.Fatalf("create stale issue: %v", err)
	}
	if err := db.Create(&duplicateIssueA).Error; err != nil {
		t.Fatalf("create duplicate issue A: %v", err)
	}
	if err := db.Create(&duplicateIssueB).Error; err != nil {
		t.Fatalf("create duplicate issue B: %v", err)
	}
	if err := db.Create(&currentIssueB).Error; err != nil {
		t.Fatalf("create current issue B: %v", err)
	}

	service := NewCatalogService(db)
	issues, err := service.ListIssues(context.Background(), IssueListFilters{View: "latest"})
	if err != nil {
		t.Fatalf("ListIssues returned error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 latest unique issues, got %d", len(issues))
	}
	for _, issue := range issues {
		if issue.Message == "stale issue" {
			t.Fatal("stale issue from old scan job should not be returned")
		}
	}
}
