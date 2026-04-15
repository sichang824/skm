package seed

import (
	"backend-go/internal/models"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

type Result struct {
	Created  int
	Existing int
	Missing  int
	Messages []string
}

type providerSpec struct {
	Name        string
	Type        string
	Icon        string
	RelativeDir string
	Priority    int
	Description string
}

var defaultProviderSpecs = []providerSpec{
	{
		Name:        "Workbuddy Skills",
		Type:        "workbuddy",
		Icon:        "github_copilot",
		RelativeDir: filepath.Join(".workbuddy", "skills"),
		Priority:    400,
		Description: "Default Workbuddy skill workspace",
	},
	{
		Name:        "Workspace Skills",
		Type:        "workspace",
		Icon:        "visual_studio_code",
		RelativeDir: filepath.Join("Workspace", "skills"),
		Priority:    350,
		Description: "Local workspace skill directory",
	},
	{
		Name:        "Agents Global",
		Type:        "global",
		Icon:        "github_copilot",
		RelativeDir: filepath.Join(".agents", "skills"),
		Priority:    300,
		Description: "Global Copilot agent skills",
	},
	{
		Name:        "Cursor Skills",
		Type:        "cursor",
		Icon:        "cursor",
		RelativeDir: filepath.Join(".cursor", "skills"),
		Priority:    200,
		Description: "Cursor local skill directory",
	},
	{
		Name:        "Codex Skills",
		Type:        "codex",
		Icon:        "codex_openai",
		RelativeDir: filepath.Join(".codex", "skills"),
		Priority:    100,
		Description: "Codex local skill directory",
	},
}

func SeedDefaultProviders(db *gorm.DB) (*Result, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home: %w", err)
	}
	return SeedDefaultProvidersForHome(db, homeDir)
}

func SeedDefaultProvidersForHome(db *gorm.DB, homeDir string) (*Result, error) {
	result := &Result{}

	for _, spec := range defaultProviderSpecs {
		rootPath := filepath.Clean(filepath.Join(homeDir, spec.RelativeDir))
		info, err := os.Stat(rootPath)
		if err != nil {
			if os.IsNotExist(err) {
				result.Missing++
				result.Messages = append(result.Messages, fmt.Sprintf("skip missing provider path: %s", rootPath))
				continue
			}
			return nil, fmt.Errorf("stat provider path %s: %w", rootPath, err)
		}
		if !info.IsDir() {
			result.Missing++
			result.Messages = append(result.Messages, fmt.Sprintf("skip non-directory provider path: %s", rootPath))
			continue
		}

		existing, err := findExistingProvider(db, spec.Name, rootPath)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			result.Existing++
			result.Messages = append(result.Messages, fmt.Sprintf("provider already exists: %s (%s)", existing.Name, existing.RootPath))
			continue
		}

		provider := models.Provider{
			Name:           spec.Name,
			Type:           spec.Type,
			Icon:           spec.Icon,
			RootPath:       rootPath,
			Enabled:        true,
			Priority:       spec.Priority,
			ScanMode:       "recursive",
			Description:    spec.Description,
			LastScanStatus: "never",
		}
		if err := db.Create(&provider).Error; err != nil {
			return nil, fmt.Errorf("create provider %s: %w", spec.Name, err)
		}

		result.Created++
		result.Messages = append(result.Messages, fmt.Sprintf("seeded provider: %s (%s)", provider.Name, provider.RootPath))
	}

	return result, nil
}

func findExistingProvider(db *gorm.DB, name, rootPath string) (*models.Provider, error) {
	var provider models.Provider
	err := db.Where("name = ? OR root_path = ?", name, rootPath).First(&provider).Error
	if err == nil {
		return &provider, nil
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return nil, fmt.Errorf("lookup existing provider %s: %w", name, err)
}
