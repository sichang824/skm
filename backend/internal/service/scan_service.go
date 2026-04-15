package service

import (
	"backend-go/internal/models"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ScanService struct {
	db *gorm.DB
}

type ScanRunResult struct {
	Jobs []models.ScanJob `json:"jobs"`
}

type discoveredSkill struct {
	RootPath       string
	SkillMdPath    string
	DirectoryName  string
	Name           string
	Slug           string
	Category       string
	Tags           []string
	Summary        string
	Status         string
	ContentHash    string
	LastModifiedAt *time.Time
	RawMarkdown    string
	BodyMarkdown   string
	Frontmatter    map[string]any
	IssueCodes     []string
}

type discoveredIssue struct {
	RootPath     string
	RelativePath string
	Code         string
	Severity     string
	Message      string
	Details      map[string]any
	SkillRoot    string
}

func NewScanService(db *gorm.DB) *ScanService {
	return &ScanService{db: db}
}

func (s *ScanService) ScanAllProviders(ctx context.Context) (*ScanRunResult, error) {
	var providers []models.Provider
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Order("priority DESC, name ASC").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("list enabled providers for scan: %w", err)
	}
	jobs := make([]models.ScanJob, 0, len(providers))
	for _, provider := range providers {
		job, err := s.scanProvider(ctx, &provider)
		if err != nil {
			return nil, wrapProviderError(&provider, "scan provider", err)
		}
		jobs = append(jobs, *job)
	}
	return &ScanRunResult{Jobs: jobs}, nil
}

func (s *ScanService) ScanProviderByZid(ctx context.Context, zid string) (*models.ScanJob, error) {
	var provider models.Provider
	if err := s.db.WithContext(ctx).Where("zid = ?", zid).First(&provider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, fmt.Errorf("load provider %s for scan: %w", zid, err)
	}
	return s.scanProvider(ctx, &provider)
}

func (s *ScanService) scanProvider(ctx context.Context, provider *models.Provider) (*models.ScanJob, error) {
	providerID := provider.ID
	job := &models.ScanJob{
		ProviderID: &providerID,
		Scope:      "provider",
		StartedAt:  time.Now(),
		Status:     "running",
		LogLines:   []string{fmt.Sprintf("scan started for provider %s", provider.Name)},
	}
	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, wrapProviderError(provider, "create scan job", err)
	}

	discoveredSkills, discoveredIssues, scanErr := discoverProvider(provider)
	finishedAt := time.Now()
	job.FinishedAt = &finishedAt
	if scanErr != nil {
		job.Status = "failed"
		job.LogLines = append(job.LogLines, scanErr.Error())
		provider.LastScannedAt = &finishedAt
		provider.LastScanStatus = "failed"
		provider.LastScanSummary = scanErr.Error()
		if err := s.db.WithContext(ctx).Save(job).Error; err != nil {
			return nil, wrapProviderError(provider, "save failed scan job", err)
		}
		if err := s.db.WithContext(ctx).Save(provider).Error; err != nil {
			return nil, wrapProviderError(provider, "save failed provider state", err)
		}
		return nil, wrapProviderError(provider, "discover provider content", scanErr)
	}

	conflictCount, addedCount, removedCount, changedCount, invalidCount, persistErr := s.persistScan(ctx, provider, job, discoveredSkills, discoveredIssues)
	if persistErr != nil {
		return nil, wrapProviderError(provider, "persist scan results", persistErr)
	}

	job.Status = "completed"
	job.AddedCount = addedCount
	job.RemovedCount = removedCount
	job.ChangedCount = changedCount
	job.InvalidCount = invalidCount
	job.ConflictCount = conflictCount
	job.LogLines = append(job.LogLines,
		fmt.Sprintf("skills discovered=%d", len(discoveredSkills)),
		fmt.Sprintf("issues detected=%d", len(discoveredIssues)),
	)
	provider.LastScannedAt = &finishedAt
	provider.LastScanStatus = "completed"
	provider.LastScanSummary = fmt.Sprintf("added=%d removed=%d changed=%d invalid=%d conflicts=%d", addedCount, removedCount, changedCount, invalidCount, conflictCount)

	if err := s.db.WithContext(ctx).Save(job).Error; err != nil {
		return nil, wrapProviderError(provider, "save completed scan job", err)
	}
	if err := s.db.WithContext(ctx).Save(provider).Error; err != nil {
		return nil, wrapProviderError(provider, "save completed provider state", err)
	}

	return job, nil
}

func (s *ScanService) persistScan(ctx context.Context, provider *models.Provider, job *models.ScanJob, discoveredSkills []discoveredSkill, discoveredIssues []discoveredIssue) (int, int, int, int, int, error) {
	var conflictCount int
	var addedCount int
	var removedCount int
	var changedCount int
	var invalidCount int

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing []models.Skill
		if err := tx.Where("provider_id = ?", provider.ID).Find(&existing).Error; err != nil {
			return fmt.Errorf("load existing skills: %w", err)
		}
		existingByRoot := make(map[string]models.Skill, len(existing))
		for _, skill := range existing {
			existingByRoot[skill.RootPath] = skill
		}

		discoveredRoots := make([]string, 0, len(discoveredSkills))
		skillIDsByRoot := make(map[string]uint, len(discoveredSkills))
		for _, discovered := range discoveredSkills {
			discoveredRoots = append(discoveredRoots, discovered.RootPath)
			record, ok := existingByRoot[discovered.RootPath]
			if !ok {
				record = models.Skill{ProviderID: provider.ID}
				addedCount++
			} else if skillChanged(record, discovered) {
				changedCount++
			}
			record.Name = discovered.Name
			record.Slug = discovered.Slug
			record.DirectoryName = discovered.DirectoryName
			record.RootPath = discovered.RootPath
			record.SkillMdPath = discovered.SkillMdPath
			record.Category = discovered.Category
			record.Tags = discovered.Tags
			record.Summary = discovered.Summary
			record.Status = discovered.Status
			record.ContentHash = discovered.ContentHash
			record.LastModifiedAt = discovered.LastModifiedAt
			record.LastScannedAt = time.Now()
			record.RawMarkdown = discovered.RawMarkdown
			record.BodyMarkdown = discovered.BodyMarkdown
			record.Frontmatter = discovered.Frontmatter
			record.IssueCodes = discovered.IssueCodes
			if slices.Contains(discovered.IssueCodes, "frontmatter_parse_failed") || slices.Contains(discovered.IssueCodes, "missing_name") || slices.Contains(discovered.IssueCodes, "name_directory_mismatch") {
				invalidCount++
			}
			if err := tx.Save(&record).Error; err != nil {
				return wrapDiscoveredSkillError(discovered, "save skill record", err)
			}
			skillIDsByRoot[record.RootPath] = record.ID
		}

		for _, skill := range existing {
			if !slices.Contains(discoveredRoots, skill.RootPath) {
				if err := tx.Delete(&skill).Error; err != nil {
					return fmt.Errorf("delete missing skill root=%s: %w", skill.RootPath, err)
				}
				removedCount++
			}
		}

		for _, issue := range discoveredIssues {
			providerID := provider.ID
			record := models.ScanIssue{
				ScanJobID:    job.ID,
				ProviderID:   &providerID,
				RootPath:     issue.RootPath,
				RelativePath: issue.RelativePath,
				Code:         issue.Code,
				Severity:     issue.Severity,
				Message:      issue.Message,
				Details:      issue.Details,
			}
			if issue.SkillRoot != "" {
				if skillID, ok := skillIDsByRoot[issue.SkillRoot]; ok {
					record.SkillID = &skillID
				}
			}
			if err := tx.Create(&record).Error; err != nil {
				return wrapDiscoveredIssueError(issue, "create scan issue", err)
			}
		}

		var err error
		conflictCount, err = rebuildConflictState(tx)
		if err != nil {
			return fmt.Errorf("rebuild conflict state: %w", err)
		}
		return err
	})

	return conflictCount, addedCount, removedCount, changedCount, invalidCount, err
}

func discoverProvider(provider *models.Provider) ([]discoveredSkill, []discoveredIssue, error) {
	skillRoots, issues, err := collectSkillRoots(provider)
	if err != nil {
		return nil, nil, wrapProviderError(provider, "collect skill roots", err)
	}
	results := make([]discoveredSkill, 0, len(skillRoots))
	for _, dirPath := range skillRoots {
		directoryName := filepath.Base(dirPath)
		skillMdPath := filepath.Join(dirPath, "SKILL.md")
		content, err := os.ReadFile(skillMdPath)
		if err != nil {
			issues = append(issues, discoveredIssue{
				RootPath:     dirPath,
				RelativePath: "SKILL.md",
				Code:         "skill_md_read_failed",
				Severity:     "error",
				Message:      "failed to read SKILL.md",
				Details:      map[string]any{"error": err.Error()},
				SkillRoot:    dirPath,
			})
			continue
		}

		parsed, parseErr := parseSkillDocument(string(content))
		status := "ready"
		issueCodes := make([]string, 0)
		if parseErr != nil {
			status = "invalid"
			issueCodes = append(issueCodes, "frontmatter_parse_failed")
			issues = append(issues, discoveredIssue{
				RootPath:     dirPath,
				RelativePath: "SKILL.md",
				Code:         "frontmatter_parse_failed",
				Severity:     "error",
				Message:      parseErr.Error(),
				SkillRoot:    dirPath,
			})
			parsed = &ParsedSkillDocument{
				Frontmatter: map[string]any{},
				Body:        string(content),
				Summary:     summarizeBody(string(content)),
				Hash:        hashContent(string(content)),
			}
		}

		name := parsed.Name
		if name == "" {
			name = directoryName
			status = "invalid"
			issueCodes = append(issueCodes, "missing_name")
			issues = append(issues, discoveredIssue{
				RootPath:     dirPath,
				RelativePath: "SKILL.md",
				Code:         "missing_name",
				Severity:     "error",
				Message:      "frontmatter is missing name",
				SkillRoot:    dirPath,
			})
		}

		expectedDirName := slugify(name)
		if expectedDirName != "" && !strings.EqualFold(expectedDirName, directoryName) {
			status = "invalid"
			issueCodes = append(issueCodes, "name_directory_mismatch")
			issues = append(issues, discoveredIssue{
				RootPath:  dirPath,
				Code:      "name_directory_mismatch",
				Severity:  "warning",
				Message:   "directory name does not match skill name",
				Details:   map[string]any{"expectedDirectory": expectedDirName, "actualDirectory": directoryName},
				SkillRoot: dirPath,
			})
		}

		info, err := os.Stat(skillMdPath)
		if err != nil {
			return nil, nil, wrapDiscoveredSkillError(discoveredSkill{RootPath: dirPath, SkillMdPath: skillMdPath}, "stat skill document", err)
		}
		modifiedAt := info.ModTime()
		results = append(results, discoveredSkill{
			RootPath:       dirPath,
			SkillMdPath:    skillMdPath,
			DirectoryName:  directoryName,
			Name:           name,
			Slug:           slugify(name),
			Category:       parsed.Category,
			Tags:           parsed.Tags,
			Summary:        parsed.Summary,
			Status:         status,
			ContentHash:    parsed.Hash,
			LastModifiedAt: &modifiedAt,
			RawMarkdown:    string(content),
			BodyMarkdown:   parsed.Body,
			Frontmatter:    parsed.Frontmatter,
			IssueCodes:     issueCodes,
		})
	}
	return results, issues, nil
}

func wrapProviderError(provider *models.Provider, action string, err error) error {
	if err == nil {
		return nil
	}
	if provider == nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	return fmt.Errorf("%s provider zid=%s name=%s root=%s: %w", action, provider.Zid, provider.Name, provider.RootPath, err)
}

func wrapDiscoveredSkillError(skill discoveredSkill, action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s root=%s skill_md=%s: %w", action, skill.RootPath, skill.SkillMdPath, err)
}

func wrapDiscoveredIssueError(issue discoveredIssue, action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s code=%s root=%s relative_path=%s: %w", action, issue.Code, issue.RootPath, issue.RelativePath, err)
}

func collectSkillRoots(provider *models.Provider) ([]string, []discoveredIssue, error) {
	scanMode := strings.ToLower(strings.TrimSpace(provider.ScanMode))
	if scanMode == "" {
		scanMode = "recursive"
	}
	if scanMode == "shallow" {
		return collectShallowSkillRoots(provider.RootPath)
	}
	return collectRecursiveSkillRoots(provider.RootPath)
}

func collectShallowSkillRoots(rootPath string) ([]string, []discoveredIssue, error) {
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, nil, err
	}
	results := make([]string, 0, len(entries))
	issues := make([]discoveredIssue, 0)
	for _, entry := range entries {
		if isIgnoredName(entry.Name()) {
			continue
		}
		dirPath := filepath.Join(rootPath, entry.Name())
		isDir, err := isDirectoryLike(dirPath, entry)
		if err != nil || !isDir {
			continue
		}
		skillMdPath := filepath.Join(dirPath, "SKILL.md")
		if _, err := os.Stat(skillMdPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				issues = append(issues, discoveredIssue{
					RootPath: dirPath,
					Code:     "missing_skill_md",
					Severity: "error",
					Message:  "directory is missing SKILL.md",
					Details:  map[string]any{"directoryName": entry.Name()},
				})
				continue
			}
			return nil, nil, err
		}
		results = append(results, dirPath)
	}
	sort.Strings(results)
	return results, issues, nil
}

func collectRecursiveSkillRoots(rootPath string) ([]string, []discoveredIssue, error) {
	results := make([]string, 0)
	err := filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == rootPath {
			return nil
		}
		isDir, err := isDirectoryLike(path, entry)
		if err != nil {
			return err
		}
		if !isDir {
			return nil
		}
		if isIgnoredName(entry.Name()) {
			return filepath.SkipDir
		}
		skillMdPath := filepath.Join(path, "SKILL.md")
		_, statErr := os.Stat(skillMdPath)
		if statErr == nil {
			results = append(results, path)
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		if !entry.IsDir() {
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(results)
	return results, []discoveredIssue{}, nil
}

func isDirectoryLike(path string, entry fs.DirEntry) (bool, error) {
	if entry.IsDir() {
		return true, nil
	}
	if entry.Type()&fs.ModeSymlink == 0 {
		return false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func skillChanged(existing models.Skill, discovered discoveredSkill) bool {
	return existing.Name != discovered.Name ||
		existing.Slug != discovered.Slug ||
		existing.Category != discovered.Category ||
		existing.Summary != discovered.Summary ||
		existing.Status != discovered.Status ||
		existing.ContentHash != discovered.ContentHash ||
		!slices.Equal(existing.Tags, discovered.Tags) ||
		!slices.Equal(existing.IssueCodes, discovered.IssueCodes)
}

func isIgnoredName(name string) bool {
	switch name {
	case ".git", ".DS_Store", "node_modules":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}
