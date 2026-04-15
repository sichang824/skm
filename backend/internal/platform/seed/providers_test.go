package seed

import (
	"backend-go/internal/models"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSeedDefaultProvidersForHomeCreatesOnlyExistingDirectories(t *testing.T) {
	db, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Provider{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	existingPaths := []string{
		filepath.Join(homeDir, ".workbuddy", "skills"),
		filepath.Join(homeDir, "Workspace", "skills"),
		filepath.Join(homeDir, ".agents", "skills"),
	}
	for _, path := range existingPaths {
		if err := ensureDir(path); err != nil {
			t.Fatalf("ensure dir %s: %v", path, err)
		}
	}

	result, err := SeedDefaultProvidersForHome(db, homeDir)
	if err != nil {
		t.Fatalf("seed providers: %v", err)
	}
	if result.Created != 3 {
		t.Fatalf("expected 3 created providers, got %d", result.Created)
	}
	if result.Missing != 2 {
		t.Fatalf("expected 2 missing providers, got %d", result.Missing)
	}

	var providers []models.Provider
	if err := db.Order("priority DESC").Find(&providers).Error; err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers in db, got %d", len(providers))
	}
	if providers[0].Name != "Workbuddy Skills" {
		t.Fatalf("expected highest priority provider to be Workbuddy Skills, got %s", providers[0].Name)
	}
	if providers[1].Name != "Workspace Skills" {
		t.Fatalf("expected second provider to be Workspace Skills, got %s", providers[1].Name)
	}
	if providers[2].Name != "Agents Global" {
		t.Fatalf("expected third provider to be Agents Global, got %s", providers[2].Name)
	}
}

func TestSeedDefaultProvidersForHomeIsIdempotent(t *testing.T) {
	db, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Provider{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	homeDir := t.TempDir()
	if err := ensureDir(filepath.Join(homeDir, ".codex", "skills")); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	first, err := SeedDefaultProvidersForHome(db, homeDir)
	if err != nil {
		t.Fatalf("first seed: %v", err)
	}
	second, err := SeedDefaultProvidersForHome(db, homeDir)
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if first.Created != 1 {
		t.Fatalf("expected first run to create 1 provider, got %d", first.Created)
	}
	if second.Created != 0 {
		t.Fatalf("expected second run to create 0 providers, got %d", second.Created)
	}
	if second.Existing != 1 {
		t.Fatalf("expected second run to find 1 existing provider, got %d", second.Existing)
	}

	var count int64
	if err := db.Model(&models.Provider{}).Count(&count).Error; err != nil {
		t.Fatalf("count providers: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 provider after reseed, got %d", count)
	}
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func openTestDB(t *testing.T) (*gorm.DB, error) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "seed-test.db")
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
}
