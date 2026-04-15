package service

import (
	"backend-go/internal/models"
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListIssuesLatestDedupesAcrossLatestJobs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Provider{}, &models.Skill{}, &models.ScanJob{}, &models.ScanIssue{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

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
