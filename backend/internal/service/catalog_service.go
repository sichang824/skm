package service

import (
	"backend-go/internal/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bmatcuk/doublestar/v4"
	"gorm.io/gorm"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrSkillNotFound    = errors.New("skill not found")
	ErrScanJobNotFound  = errors.New("scan job not found")
	ErrInvalidInput     = errors.New("invalid input")
	ErrBinaryFile       = errors.New("binary file preview is not supported")
)

const (
	skillRelationFromFile = ".from"
	skillRelationToFile   = ".to"
	skillAttachModeMove   = "move"
	skillAttachModeAttach = "attach"
)

type skillRelationState struct {
	FromPath string
	To       skillToMetadata
	HasFrom  bool
	HasTo    bool
}

type skillToMetadata struct {
	Directories []string `json:"directories"`
	Include     []string `json:"include,omitempty"`
	Exclude     []string `json:"exclude,omitempty"`
	LegacyFiles []string `json:"files,omitempty"`
}

type skillCopyRules struct {
	Include []string
	Exclude []string
}

type CatalogService struct {
	db *gorm.DB
}

type ProviderInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Icon        string `json:"icon"`
	RootPath    string `json:"rootPath"`
	Enabled     bool   `json:"enabled"`
	Priority    int    `json:"priority"`
	ScanMode    string `json:"scanMode"`
	Description string `json:"description"`
}

type DashboardSummary struct {
	ProviderCount        int64 `json:"providerCount"`
	EnabledProviderCount int64 `json:"enabledProviderCount"`
	SkillCount           int64 `json:"skillCount"`
	ConflictCount        int64 `json:"conflictCount"`
	IssueCount           int64 `json:"issueCount"`
	RecentScanCount      int64 `json:"recentScanCount"`
}

type SkillListFilters struct {
	Query    string
	Provider string
	Category string
	Tag      string
	Status   string
	Conflict *bool
	Sort     string
	Grouped  bool
}

type IssueListFilters struct {
	View     string
	Provider string
	Severity string
	Code     string
}

type FileNode struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	IsDir      bool       `json:"isDir"`
	Size       int64      `json:"size,omitempty"`
	ModifiedAt *time.Time `json:"modifiedAt,omitempty"`
	Children   []FileNode `json:"children,omitempty"`
}

type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type SkillAttachInput struct {
	TargetProviderZid string `json:"targetProviderZid"`
	Mode              string `json:"mode"`
}

type SkillToInput struct {
	RootPath     string   `json:"rootPath"`
	ProviderPath string   `json:"providerPath,omitempty"`
	Directories  []string `json:"directories,omitempty"`
	Include      []string `json:"include,omitempty"`
	Exclude      []string `json:"exclude,omitempty"`
}

type SkillToResult struct {
	RootPath        string                `json:"rootPath"`
	FilePath        string                `json:"filePath"`
	Provider        *models.Provider      `json:"provider,omitempty"`
	ProviderCreated bool                  `json:"providerCreated"`
	Relation        *models.SkillRelation `json:"relation,omitempty"`
}

type SkillAttachScanJob struct {
	ProviderZid string         `json:"providerZid"`
	Job         models.ScanJob `json:"job"`
}

type SkillAttachResult struct {
	SkillZid       string               `json:"skillZid"`
	Mode           string               `json:"mode"`
	SourceProvider models.Provider      `json:"sourceProvider"`
	TargetProvider models.Provider      `json:"targetProvider"`
	SourcePath     string               `json:"sourcePath"`
	TargetPath     string               `json:"targetPath"`
	Jobs           []SkillAttachScanJob `json:"jobs,omitempty"`
}

type SkillDeleteResult struct {
	SkillZid    string          `json:"skillZid"`
	Provider    models.Provider `json:"provider"`
	DeletedPath string          `json:"deletedPath"`
	Deleted     bool            `json:"deleted"`
	Forced      bool            `json:"forced"`
	DeleteMode  string          `json:"deleteMode"`
	CopyCount   int             `json:"copyCount,omitempty"`
	Job         *models.ScanJob `json:"job,omitempty"`
}

type SkillDeleteInput struct {
	Force bool `json:"force"`
}

type SkillSyncResult struct {
	SkillZid   string          `json:"skillZid"`
	Provider   models.Provider `json:"provider"`
	SourcePath string          `json:"sourcePath"`
	TargetPath string          `json:"targetPath"`
	Synced     bool            `json:"synced"`
	Job        *models.ScanJob `json:"job,omitempty"`
}

func NewCatalogService(db *gorm.DB) *CatalogService {
	return &CatalogService{db: db}
}

func (s *CatalogService) Dashboard(ctx context.Context) (*DashboardSummary, error) {
	var summary DashboardSummary

	if err := s.db.WithContext(ctx).Model(&models.Provider{}).Count(&summary.ProviderCount).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&models.Provider{}).Where("enabled = ?", true).Count(&summary.EnabledProviderCount).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).
		Model(&models.Skill{}).
		Joins("JOIN providers ON providers.id = skills.provider_id").
		Where("providers.enabled = ?", true).
		Count(&summary.SkillCount).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).
		Model(&models.Skill{}).
		Joins("JOIN providers ON providers.id = skills.provider_id").
		Where("providers.enabled = ? AND skills.is_conflict = ?", true, true).
		Count(&summary.ConflictCount).Error; err != nil {
		return nil, err
	}
	issueCount, err := s.countIssues(ctx, IssueListFilters{View: "latest"})
	if err != nil {
		return nil, err
	}
	summary.IssueCount = issueCount
	cutoff := time.Now().Add(-24 * time.Hour)
	if err := s.db.WithContext(ctx).Model(&models.ScanJob{}).Where("started_at >= ?", cutoff).Count(&summary.RecentScanCount).Error; err != nil {
		return nil, err
	}

	return &summary, nil
}

func (s *CatalogService) ListProviders(ctx context.Context) ([]models.Provider, error) {
	var providers []models.Provider
	if err := s.db.WithContext(ctx).Order("priority DESC, name ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

func (s *CatalogService) GetProvider(ctx context.Context, zid string) (*models.Provider, error) {
	var provider models.Provider
	if err := s.db.WithContext(ctx).Where("zid = ?", zid).First(&provider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return &provider, nil
}

func (s *CatalogService) CreateProvider(ctx context.Context, input ProviderInput) (*models.Provider, error) {
	provider, err := s.normalizeProviderInput(ctx, nil, input)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(provider).Error; err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *CatalogService) UpdateProvider(ctx context.Context, zid string, input ProviderInput) (*models.Provider, error) {
	existing, err := s.GetProvider(ctx, zid)
	if err != nil {
		return nil, err
	}
	provider, err := s.normalizeProviderInput(ctx, existing, input)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Save(provider).Error; err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *CatalogService) DeleteProvider(ctx context.Context, zid string) error {
	provider, err := s.GetProvider(ctx, zid)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Delete(provider).Error
}

func (s *CatalogService) ListSkills(ctx context.Context, filters SkillListFilters) ([]models.Skill, error) {
	query := s.db.WithContext(ctx).
		Model(&models.Skill{}).
		Preload("Provider").
		Joins("JOIN providers ON providers.id = skills.provider_id").
		Where("providers.enabled = ?", true)

	if filters.Query != "" {
		like := "%" + strings.ToLower(strings.TrimSpace(filters.Query)) + "%"
		query = query.Where("LOWER(skills.name) LIKE ? OR LOWER(skills.summary) LIKE ?", like, like)
	}
	if filters.Provider != "" {
		query = query.Where("providers.zid = ? OR providers.name = ?", filters.Provider, filters.Provider)
	}
	if filters.Category != "" {
		query = query.Where("skills.category = ?", filters.Category)
	}
	if filters.Status != "" {
		query = query.Where("skills.status = ?", filters.Status)
	}
	if filters.Conflict != nil {
		query = query.Where("skills.is_conflict = ?", *filters.Conflict)
	}

	orderBy := map[string]string{
		"name":        "skills.name ASC",
		"provider":    "providers.name ASC, skills.name ASC",
		"status":      "skills.status ASC, skills.name ASC",
		"lastScanned": "skills.last_scanned_at DESC, skills.name ASC",
	}[filters.Sort]
	if orderBy == "" {
		orderBy = "skills.name ASC"
	}

	var skills []models.Skill
	if err := query.Order(orderBy).Find(&skills).Error; err != nil {
		return nil, err
	}
	if filters.Tag != "" {
		filtered := make([]models.Skill, 0, len(skills))
		for _, skill := range skills {
			for _, tag := range skill.Tags {
				if strings.EqualFold(tag, filters.Tag) {
					filtered = append(filtered, skill)
					break
				}
			}
		}
		skills = filtered
	}
	for index := range skills {
		skills[index].Relation = readSkillRelationForDisplay(skills[index].RootPath)
	}
	if filters.Grouped {
		return groupSkillsForList(skills), nil
	}
	return skills, nil
}

func (s *CatalogService) GetSkill(ctx context.Context, zid string) (*models.Skill, error) {
	var skill models.Skill
	if err := s.db.WithContext(ctx).
		Preload("Provider").
		Joins("JOIN providers ON providers.id = skills.provider_id").
		Where("skills.zid = ? AND providers.enabled = ?", zid, true).
		First(&skill).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSkillNotFound
		}
		return nil, err
	}
	skill.Relation = readSkillRelationForDisplay(skill.RootPath)
	return &skill, nil
}

func (s *CatalogService) GetSkillFiles(ctx context.Context, zid string) ([]FileNode, error) {
	skill, err := s.GetSkill(ctx, zid)
	if err != nil {
		return nil, err
	}
	return listFileNodes(skill.RootPath, skill.RootPath)
}

func (s *CatalogService) GetSkillFileContent(ctx context.Context, zid, relativePath string) (*FileContent, error) {
	skill, err := s.GetSkill(ctx, zid)
	if err != nil {
		return nil, err
	}
	absPath, err := safeJoin(skill.RootPath, relativePath)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid file path", ErrInvalidInput)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), rune(0)) {
		return nil, ErrBinaryFile
	}
	return &FileContent{Path: relativePath, Content: string(data)}, nil
}

func (s *CatalogService) AttachSkill(ctx context.Context, skillZid string, input SkillAttachInput) (*SkillAttachResult, error) {
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode != skillAttachModeMove && mode != skillAttachModeAttach {
		return nil, fmt.Errorf("%w: mode must be move or attach", ErrInvalidInput)
	}
	if strings.TrimSpace(input.TargetProviderZid) == "" {
		return nil, fmt.Errorf("%w: targetProviderZid is required", ErrInvalidInput)
	}

	skill, err := s.GetSkill(ctx, skillZid)
	if err != nil {
		return nil, err
	}
	targetProvider, err := s.GetProvider(ctx, input.TargetProviderZid)
	if err != nil {
		return nil, err
	}
	if !targetProvider.Enabled {
		return nil, fmt.Errorf("%w: target provider must be enabled", ErrInvalidInput)
	}
	if skill.Provider.Zid == targetProvider.Zid {
		return nil, fmt.Errorf("%w: skill already belongs to target provider", ErrInvalidInput)
	}

	sourcePath := filepath.Clean(skill.RootPath)
	if !pathWithinRoot(skill.Provider.RootPath, sourcePath) {
		return nil, fmt.Errorf("%w: skill rootPath is outside source provider root", ErrInvalidInput)
	}
	targetPath, err := safeJoin(targetProvider.RootPath, skill.DirectoryName)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid target path", ErrInvalidInput)
	}
	if _, statErr := os.Lstat(targetPath); statErr == nil {
		if mode == skillAttachModeMove {
			return nil, fmt.Errorf("%w: target path already exists", ErrInvalidInput)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, statErr
	}

	if mode == skillAttachModeMove {
		if err := moveDirectory(sourcePath, targetPath); err != nil {
			return nil, err
		}
		if err := updateRelationsAfterMove(sourcePath, targetPath); err != nil {
			return nil, err
		}
	} else {
		sourceRelation, err := readSkillRelationState(sourcePath)
		if err != nil {
			return nil, err
		}
		if sourceRelation.HasFrom {
			return nil, fmt.Errorf("%w: source skill already has .from metadata", ErrInvalidInput)
		}

		targetRelation, err := readSkillRelationState(targetPath)
		if err != nil {
			return nil, err
		}
		if targetRelation.HasTo {
			return nil, fmt.Errorf("%w: target skill already has .to metadata", ErrInvalidInput)
		}

		if err := copyDirectory(sourcePath, targetPath, copyDirectoryOptions{Overwrite: true, SkipRootFiles: map[string]struct{}{
			skillRelationFromFile: {},
			skillRelationToFile:   {},
		}, Rules: copyRulesFromMetadata(sourceRelation.To)}); err != nil {
			return nil, err
		}
		sourceRelation.To.Directories = append(sourceRelation.To.Directories, targetPath)
		if err := writeSkillToMetadata(sourcePath, sourceRelation.To); err != nil {
			return nil, err
		}
		if err := writeSkillFromMetadata(targetPath, sourcePath); err != nil {
			return nil, err
		}
	}

	return &SkillAttachResult{
		SkillZid:       skill.Zid,
		Mode:           mode,
		SourceProvider: skill.Provider,
		TargetProvider: *targetProvider,
		SourcePath:     sourcePath,
		TargetPath:     targetPath,
	}, nil
}

func (s *CatalogService) DeleteSkill(ctx context.Context, skillZid string, input SkillDeleteInput) (*SkillDeleteResult, error) {
	skill, err := s.GetSkill(ctx, skillZid)
	if err != nil {
		return nil, err
	}

	deletePath := filepath.Clean(skill.RootPath)
	providerRoot := filepath.Clean(skill.Provider.RootPath)
	relation, err := readSkillRelationState(deletePath)
	if err != nil {
		return nil, err
	}
	if deletePath == providerRoot {
		return nil, fmt.Errorf("%w: deleting provider root is not allowed", ErrInvalidInput)
	}
	if !pathWithinRoot(providerRoot, deletePath) {
		return nil, fmt.Errorf("%w: skill rootPath is outside provider root", ErrInvalidInput)
	}
	deleteMode := "plain"
	copyCount := 0
	if relation.HasFrom {
		deleteMode = "attached-copy"
		if err := removeDirectoryFromSourceRelation(relation.FromPath, deletePath); err != nil {
			return nil, err
		}
	} else if relation.HasTo {
		copyCount = len(relation.To.Directories)
		if copyCount > 0 && !input.Force {
			return nil, fmt.Errorf("%w: skill has %d attached copies; use force delete to remove the source only", ErrInvalidInput, copyCount)
		}
		if copyCount > 0 {
			deleteMode = "source-force"
		} else {
			deleteMode = "source"
		}
	}
	if err := os.RemoveAll(deletePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return &SkillDeleteResult{
		SkillZid:    skill.Zid,
		Provider:    skill.Provider,
		DeletedPath: deletePath,
		Deleted:     true,
		Forced:      input.Force,
		DeleteMode:  deleteMode,
		CopyCount:   copyCount,
	}, nil
}

func (s *CatalogService) SyncSkill(ctx context.Context, skillZid string) (*SkillSyncResult, error) {
	skill, err := s.GetSkill(ctx, skillZid)
	if err != nil {
		return nil, err
	}

	targetPath := filepath.Clean(skill.RootPath)
	providerRoot := filepath.Clean(skill.Provider.RootPath)
	if !pathWithinRoot(providerRoot, targetPath) {
		return nil, fmt.Errorf("%w: skill rootPath is outside provider root", ErrInvalidInput)
	}

	relation, err := readSkillRelationState(targetPath)
	if err != nil {
		return nil, err
	}
	if !relation.HasFrom || strings.TrimSpace(relation.FromPath) == "" {
		return nil, fmt.Errorf("%w: skill is not an attached copy", ErrInvalidInput)
	}
	sourcePath := filepath.Clean(relation.FromPath)
	if sourcePath == targetPath {
		return nil, fmt.Errorf("%w: sourcePath must differ from targetPath", ErrInvalidInput)
	}
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	if !sourceInfo.IsDir() {
		return nil, fmt.Errorf("%w: sourcePath must be a directory", ErrInvalidInput)
	}
	sourceRelation, err := readSkillRelationState(sourcePath)
	if err != nil {
		return nil, err
	}
	if sourceRelation.HasFrom {
		return nil, fmt.Errorf("%w: source skill already has .from metadata", ErrInvalidInput)
	}

	if err := clearDirectoryContents(targetPath, map[string]struct{}{skillRelationFromFile: {}}); err != nil {
		return nil, err
	}
	if err := copyDirectory(sourcePath, targetPath, copyDirectoryOptions{Overwrite: true, SkipRootFiles: map[string]struct{}{
		skillRelationFromFile: {},
		skillRelationToFile:   {},
	}, Rules: copyRulesFromMetadata(sourceRelation.To)}); err != nil {
		return nil, err
	}
	if err := writeSkillFromMetadata(targetPath, sourcePath); err != nil {
		return nil, err
	}
	sourceRelation.To.Directories = append(sourceRelation.To.Directories, targetPath)
	if err := writeSkillToMetadata(sourcePath, sourceRelation.To); err != nil {
		return nil, err
	}

	return &SkillSyncResult{
		SkillZid:   skill.Zid,
		Provider:   skill.Provider,
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Synced:     true,
	}, nil
}

func (s *CatalogService) ConfigureSkillTo(ctx context.Context, input SkillToInput) (*SkillToResult, error) {
	rootPath := strings.TrimSpace(input.RootPath)
	if rootPath == "" {
		return nil, fmt.Errorf("%w: rootPath is required", ErrInvalidInput)
	}
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid rootPath", ErrInvalidInput)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: rootPath must be a directory", ErrInvalidInput)
	}
	if _, err := os.Stat(filepath.Join(absRoot, "SKILL.md")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: SKILL.md not found in rootPath", ErrInvalidInput)
		}
		return nil, err
	}

	state, err := readSkillRelationState(absRoot)
	if err != nil {
		return nil, err
	}
	if state.HasFrom {
		return nil, fmt.Errorf("%w: attached copies cannot own .to metadata", ErrInvalidInput)
	}

	metadata := state.To
	providerPath := strings.TrimSpace(input.ProviderPath)
	if providerPath == "" {
		providerPath = absRoot
	}
	absProviderPath, err := filepath.Abs(providerPath)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid providerPath", ErrInvalidInput)
	}
	if err := validateDirectory(absProviderPath); err != nil {
		return nil, fmt.Errorf("%w: providerPath is not accessible", ErrInvalidInput)
	}
	absProviderPath = filepath.Clean(absProviderPath)
	if !pathWithinRoot(absProviderPath, absRoot) {
		return nil, fmt.Errorf("%w: providerPath must be the current directory or a parent directory", ErrInvalidInput)
	}

	provider, created, err := s.ensureProviderForSkillRoot(ctx, absRoot, absProviderPath)
	if err != nil {
		return nil, err
	}
	result := &SkillToResult{
		RootPath:        absRoot,
		FilePath:        filepath.Join(absRoot, skillRelationToFile),
		Provider:        provider,
		ProviderCreated: created,
	}

	for _, directory := range input.Directories {
		trimmed := strings.TrimSpace(directory)
		if trimmed == "" {
			continue
		}
		absDirectory, err := filepath.Abs(trimmed)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid directory %q", ErrInvalidInput, directory)
		}
		metadata.Directories = append(metadata.Directories, absDirectory)
	}
	if len(input.Include) > 0 {
		metadata.Include = append([]string{}, input.Include...)
	}
	if len(input.Exclude) > 0 {
		metadata.Exclude = append([]string{}, input.Exclude...)
	}
	metadata = normalizeSkillToMetadata(metadata)
	if err := writeSkillToMetadata(absRoot, metadata); err != nil {
		return nil, err
	}
	result.Relation = &models.SkillRelation{
		Mode:        "to",
		Directories: append([]string{}, metadata.Directories...),
		Include:     append([]string{}, metadata.Include...),
		Exclude:     append([]string{}, metadata.Exclude...),
	}
	return result, nil
}

func (s *CatalogService) ensureProviderForSkillRoot(ctx context.Context, skillRoot, providerPath string) (*models.Provider, bool, error) {
	providers, err := s.ListProviders(ctx)
	if err != nil {
		return nil, false, err
	}
	for index := range providers {
		if pathWithinRoot(providers[index].RootPath, skillRoot) {
			return &providers[index], false, nil
		}
	}

	providerName, err := s.nextProviderName(ctx, filepath.Base(providerPath))
	if err != nil {
		return nil, false, err
	}
	provider, err := s.CreateProvider(ctx, ProviderInput{
		Name:     providerName,
		Type:     "workspace",
		RootPath: providerPath,
		Enabled:  true,
		Priority: 100,
		ScanMode: "recursive",
	})
	if err != nil {
		return nil, false, err
	}
	return provider, true, nil
}

func (s *CatalogService) nextProviderName(ctx context.Context, base string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" || base == string(filepath.Separator) || base == "." {
		base = "provider"
	}

	providers, err := s.ListProviders(ctx)
	if err != nil {
		return "", err
	}
	used := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		used[strings.ToLower(provider.Name)] = struct{}{}
	}
	if _, exists := used[strings.ToLower(base)]; !exists {
		return base, nil
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s (%d)", base, index)
		if _, exists := used[strings.ToLower(candidate)]; !exists {
			return candidate, nil
		}
	}
}

func (s *CatalogService) ListScanJobs(ctx context.Context) ([]models.ScanJob, error) {
	var jobs []models.ScanJob
	if err := s.db.WithContext(ctx).Preload("Provider").Order("started_at DESC").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *CatalogService) GetScanJob(ctx context.Context, zid string) (*models.ScanJob, []models.ScanIssue, error) {
	var job models.ScanJob
	if err := s.db.WithContext(ctx).Preload("Provider").Where("zid = ?", zid).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrScanJobNotFound
		}
		return nil, nil, err
	}
	var issues []models.ScanIssue
	if err := s.db.WithContext(ctx).Preload("Provider").Preload("Skill").Where("scan_job_id = ?", job.ID).Order("created_at ASC").Find(&issues).Error; err != nil {
		return nil, nil, err
	}
	return &job, issues, nil
}

func (s *CatalogService) ListIssues(ctx context.Context, filters IssueListFilters) ([]models.ScanIssue, error) {
	view := strings.ToLower(strings.TrimSpace(filters.View))
	if view == "latest" {
		jobIDs, err := s.latestScanJobIDs(ctx, filters.Provider)
		if err != nil {
			return nil, err
		}
		if len(jobIDs) == 0 {
			return []models.ScanIssue{}, nil
		}
		query := s.db.WithContext(ctx).
			Preload("Provider").
			Preload("Skill").
			Where("scan_job_id IN ?", jobIDs).
			Order("created_at DESC")
		query = applyIssueFilters(query, filters, false)
		var issues []models.ScanIssue
		if err := query.Find(&issues).Error; err != nil {
			return nil, err
		}
		return dedupeIssues(issues), nil
	}

	query := s.db.WithContext(ctx).
		Preload("Provider").
		Preload("Skill").
		Order("created_at DESC")
	query = applyIssueFilters(query, filters, true)
	var issues []models.ScanIssue
	if err := query.Find(&issues).Error; err != nil {
		return nil, err
	}
	return issues, nil
}

func (s *CatalogService) ListConflicts(ctx context.Context) ([]ConflictGroup, error) {
	var skills []models.Skill
	if err := s.db.WithContext(ctx).
		Preload("Provider").
		Joins("JOIN providers ON providers.id = skills.provider_id").
		Where("providers.enabled = ?", true).
		Find(&skills).Error; err != nil {
		return nil, err
	}
	groups := buildConflictGroups(skills)
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Kind == groups[j].Kind {
			return groups[i].Key < groups[j].Key
		}
		return groups[i].Kind < groups[j].Kind
	})
	return groups, nil
}

func (s *CatalogService) normalizeProviderInput(ctx context.Context, existing *models.Provider, input ProviderInput) (*models.Provider, error) {
	name := strings.TrimSpace(input.Name)
	providerType := strings.TrimSpace(input.Type)
	icon := normalizeProviderIcon(input.Icon)
	rootPath := strings.TrimSpace(input.RootPath)
	scanMode := strings.ToLower(strings.TrimSpace(input.ScanMode))
	if scanMode == "" {
		scanMode = "recursive"
	}
	if scanMode != "shallow" && scanMode != "recursive" {
		return nil, fmt.Errorf("%w: unsupported scan mode", ErrInvalidInput)
	}
	if name == "" || providerType == "" || rootPath == "" {
		return nil, fmt.Errorf("%w: name, type and rootPath are required", ErrInvalidInput)
	}
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid rootPath", ErrInvalidInput)
	}
	if err := validateDirectory(absRoot); err != nil {
		return nil, err
	}

	provider := &models.Provider{}
	if existing != nil {
		*provider = *existing
	}
	provider.Name = name
	provider.Type = providerType
	provider.Icon = icon
	provider.RootPath = filepath.Clean(absRoot)
	provider.Enabled = input.Enabled
	provider.Priority = input.Priority
	provider.ScanMode = scanMode
	provider.Description = strings.TrimSpace(input.Description)
	if provider.Priority == 0 {
		provider.Priority = 100
	}

	var providers []models.Provider
	if err := s.db.WithContext(ctx).Find(&providers).Error; err != nil {
		return nil, err
	}
	for _, candidate := range providers {
		if existing != nil && candidate.ID == existing.ID {
			continue
		}
		if strings.EqualFold(candidate.Name, provider.Name) {
			return nil, fmt.Errorf("%w: provider name already exists", ErrInvalidInput)
		}
		if pathsOverlap(candidate.RootPath, provider.RootPath) {
			return nil, fmt.Errorf("%w: provider rootPath conflicts with existing provider %s", ErrInvalidInput, candidate.Name)
		}
	}

	return provider, nil
}

func normalizeProviderIcon(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(normalized) + 2)
	lastUnderscore := false
	for _, char := range normalized {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
			lastUnderscore = false
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastUnderscore = false
		default:
			if builder.Len() == 0 || lastUnderscore {
				continue
			}
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}

	normalized = strings.Trim(builder.String(), "_")
	if normalized == "" {
		return ""
	}
	if normalized[0] >= '0' && normalized[0] <= '9' {
		return "i_" + normalized
	}
	return normalized
}

func (s *CatalogService) countIssues(ctx context.Context, filters IssueListFilters) (int64, error) {
	issues, err := s.ListIssues(ctx, filters)
	if err != nil {
		return 0, err
	}
	return int64(len(issues)), nil
}

func (s *CatalogService) latestScanJobIDs(ctx context.Context, provider string) ([]uint, error) {
	query := s.db.WithContext(ctx).
		Model(&models.ScanJob{}).
		Select("MAX(scan_jobs.id) AS id").
		Where("scan_jobs.provider_id IS NOT NULL").
		Group("scan_jobs.provider_id")
	if provider != "" {
		query = query.
			Joins("JOIN providers ON providers.id = scan_jobs.provider_id").
			Where("providers.zid = ? OR providers.name = ?", provider, provider)
	}
	var rows []struct {
		ID uint
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	jobIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.ID != 0 {
			jobIDs = append(jobIDs, row.ID)
		}
	}
	return jobIDs, nil
}

func applyIssueFilters(query *gorm.DB, filters IssueListFilters, allowProviderJoin bool) *gorm.DB {
	if allowProviderJoin && filters.Provider != "" {
		query = query.
			Joins("LEFT JOIN providers AS issue_providers ON issue_providers.id = scan_issues.provider_id").
			Where("issue_providers.zid = ? OR issue_providers.name = ?", filters.Provider, filters.Provider)
	}
	if filters.Severity != "" {
		query = query.Where("scan_issues.severity = ?", filters.Severity)
	}
	if filters.Code != "" {
		query = query.Where("scan_issues.code = ?", filters.Code)
	}
	return query
}

func dedupeIssues(issues []models.ScanIssue) []models.ScanIssue {
	seen := make(map[string]struct{}, len(issues))
	result := make([]models.ScanIssue, 0, len(issues))
	for _, issue := range issues {
		key := issueFingerprint(issue)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, issue)
	}
	return result
}

func issueFingerprint(issue models.ScanIssue) string {
	providerKey := ""
	if issue.Provider != nil {
		providerKey = issue.Provider.Zid
	}
	skillKey := ""
	if issue.Skill != nil {
		skillKey = issue.Skill.Zid
	}
	return strings.Join([]string{
		providerKey,
		skillKey,
		issue.RootPath,
		issue.RelativePath,
		issue.Code,
		issue.Severity,
		issue.Message,
	}, "|")
}

func validateDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%w: rootPath is not accessible", ErrInvalidInput)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: rootPath must be a directory", ErrInvalidInput)
	}
	if _, err := os.ReadDir(path); err != nil {
		return fmt.Errorf("%w: rootPath is not readable", ErrInvalidInput)
	}
	return nil
}

func pathsOverlap(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if left == right {
		return true
	}
	if rel, err := filepath.Rel(left, right); err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return true
	}
	if rel, err := filepath.Rel(right, left); err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return true
	}
	return false
}

func pathWithinRoot(rootPath, candidate string) bool {
	rootPath = filepath.Clean(rootPath)
	candidate = filepath.Clean(candidate)
	if rootPath == candidate {
		return true
	}
	rel, err := filepath.Rel(rootPath, candidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func safeJoin(rootPath, relativePath string) (string, error) {
	cleanRelative := filepath.Clean(relativePath)
	joined := filepath.Join(rootPath, cleanRelative)
	rel, err := filepath.Rel(rootPath, joined)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrInvalidInput
	}
	return joined, nil
}

func moveDirectory(sourcePath, targetPath string) error {
	if err := os.Rename(sourcePath, targetPath); err == nil {
		return nil
	} else if linkErr, ok := err.(*os.LinkError); !ok || linkErr.Err == nil || !strings.Contains(strings.ToLower(linkErr.Err.Error()), "cross-device") {
		return err
	}

	if err := copyDirectory(sourcePath, targetPath, copyDirectoryOptions{}); err != nil {
		_ = os.RemoveAll(targetPath)
		return err
	}
	return os.RemoveAll(sourcePath)
}

type copyDirectoryOptions struct {
	Overwrite     bool
	SkipRootFiles map[string]struct{}
	Rules         skillCopyRules
}

func clearDirectoryContents(rootPath string, skipNames map[string]struct{}) error {
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if _, skip := skipNames[entry.Name()]; skip {
			continue
		}
		if err := os.RemoveAll(filepath.Join(rootPath, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyDirectory(sourcePath, targetPath string, options copyDirectoryOptions) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: sourcePath must be a directory", ErrInvalidInput)
	}
	if existingInfo, statErr := os.Stat(targetPath); statErr == nil {
		if !existingInfo.IsDir() {
			return fmt.Errorf("%w: targetPath must be a directory", ErrInvalidInput)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
		return err
	}

	return filepath.Walk(sourcePath, func(path string, _ os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourcePath {
			return nil
		}
		rel, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		if _, skip := options.SkipRootFiles[rel]; skip {
			return nil
		}
		targetEntryPath := filepath.Join(targetPath, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		shouldCopy, err := options.Rules.shouldCopy(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		if !shouldCopy {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if options.Overwrite {
				if err := os.RemoveAll(targetEntryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(targetEntryPath), 0o755); err != nil {
				return err
			}
			return os.Symlink(linkTarget, targetEntryPath)
		}
		if err := os.MkdirAll(filepath.Dir(targetEntryPath), 0o755); err != nil {
			return err
		}
		return copyFile(path, targetEntryPath, info.Mode().Perm())
	})
}

func readSkillRelationForDisplay(rootPath string) *models.SkillRelation {
	state, err := readSkillRelationState(rootPath)
	if err != nil {
		return nil
	}
	if state.HasFrom {
		return &models.SkillRelation{Mode: "from", FromPath: state.FromPath}
	}
	if state.HasTo {
		return &models.SkillRelation{
			Mode:        "to",
			Directories: append([]string{}, state.To.Directories...),
			Include:     append([]string{}, state.To.Include...),
			Exclude:     append([]string{}, state.To.Exclude...),
		}
	}
	return nil
}

func groupSkillsForList(skills []models.Skill) []models.Skill {
	indicesByRoot := make(map[string]int, len(skills))
	for index := range skills {
		skills[index].RelatedSkills = nil
		indicesByRoot[filepath.Clean(skills[index].RootPath)] = index
	}

	nested := make(map[int]struct{}, len(skills))
	for index := range skills {
		relation := skills[index].Relation
		if relation == nil || relation.Mode != "from" {
			continue
		}
		parentIndex, ok := indicesByRoot[filepath.Clean(strings.TrimSpace(relation.FromPath))]
		if !ok {
			continue
		}
		parentRelation := skills[parentIndex].Relation
		if parentRelation == nil || parentRelation.Mode != "to" {
			continue
		}
		skills[parentIndex].RelatedSkills = append(skills[parentIndex].RelatedSkills, skills[index])
		nested[index] = struct{}{}
	}

	grouped := make([]models.Skill, 0, len(skills))
	for index := range skills {
		if _, isNested := nested[index]; isNested {
			continue
		}
		grouped = append(grouped, skills[index])
	}
	return grouped
}

func readSkillRelationState(rootPath string) (skillRelationState, error) {
	state := skillRelationState{}
	fromPath := filepath.Join(rootPath, skillRelationFromFile)
	fromData, err := os.ReadFile(fromPath)
	if err == nil {
		state.HasFrom = true
		state.FromPath = strings.TrimSpace(string(fromData))
	} else if !errors.Is(err, os.ErrNotExist) {
		return state, err
	}

	toPath := filepath.Join(rootPath, skillRelationToFile)
	toData, err := os.ReadFile(toPath)
	if err == nil {
		state.HasTo = true
		if unmarshalErr := json.Unmarshal(toData, &state.To); unmarshalErr != nil {
			return state, fmt.Errorf("%w: invalid .to metadata", ErrInvalidInput)
		}
		state.To = normalizeSkillToMetadata(state.To)
	} else if !errors.Is(err, os.ErrNotExist) {
		return state, err
	}

	if state.HasFrom && state.HasTo {
		return state, fmt.Errorf("%w: .from and .to cannot coexist", ErrInvalidInput)
	}
	return state, nil
}

func normalizeSkillToMetadata(metadata skillToMetadata) skillToMetadata {
	metadata.Directories = uniqueSortedStrings(metadata.Directories)
	metadata.Include = uniqueSortedPatterns(metadata.Include)
	metadata.Exclude = uniqueSortedPatterns(metadata.Exclude)
	if len(metadata.Include) == 0 {
		metadata.Include = []string{"README.md", "SKILL.md"}
	}
	if len(metadata.Exclude) == 0 {
		metadata.Exclude = []string{"**/.DS_Store"}
	}
	metadata.LegacyFiles = nil
	return metadata
}

func copyRulesFromMetadata(metadata skillToMetadata) skillCopyRules {
	metadata = normalizeSkillToMetadata(metadata)
	return skillCopyRules{Include: metadata.Include, Exclude: metadata.Exclude}
}

func (r skillCopyRules) shouldCopy(relativePath string) (bool, error) {
	normalizedPath := filepath.ToSlash(strings.TrimSpace(relativePath))
	if normalizedPath == "" || normalizedPath == "." {
		return false, nil
	}
	if normalizedPath == skillRelationFromFile || normalizedPath == skillRelationToFile {
		return false, nil
	}
	included := len(r.Include) == 0
	for _, pattern := range r.Include {
		matched, err := doublestar.Match(pattern, normalizedPath)
		if err != nil {
			return false, fmt.Errorf("%w: invalid include pattern %q", ErrInvalidInput, pattern)
		}
		if matched {
			included = true
			break
		}
	}
	if !included {
		return false, nil
	}
	for _, pattern := range r.Exclude {
		matched, err := doublestar.Match(pattern, normalizedPath)
		if err != nil {
			return false, fmt.Errorf("%w: invalid exclude pattern %q", ErrInvalidInput, pattern)
		}
		if matched {
			return false, nil
		}
	}
	return true, nil
}

func writeSkillFromMetadata(rootPath, fromPath string) error {
	if err := os.Remove(filepath.Join(rootPath, skillRelationToFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(filepath.Join(rootPath, skillRelationFromFile), []byte(filepath.Clean(fromPath)+"\n"), 0o644)
}

func writeSkillToMetadata(rootPath string, metadata skillToMetadata) error {
	if err := os.Remove(filepath.Join(rootPath, skillRelationFromFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	metadata = normalizeSkillToMetadata(metadata)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(rootPath, skillRelationToFile), data, 0o644)
}

func updateRelationsAfterMove(sourcePath, targetPath string) error {
	state, err := readSkillRelationState(targetPath)
	if err != nil {
		return err
	}
	if state.HasTo {
		for _, directory := range state.To.Directories {
			if err := writeSkillFromMetadata(directory, targetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	if !state.HasFrom {
		return nil
	}
	sourceState, err := readSkillRelationState(state.FromPath)
	if err != nil {
		return err
	}
	if !sourceState.HasTo {
		return nil
	}
	directories := make([]string, 0, len(sourceState.To.Directories))
	for _, directory := range sourceState.To.Directories {
		if filepath.Clean(directory) == filepath.Clean(sourcePath) {
			directories = append(directories, targetPath)
			continue
		}
		directories = append(directories, directory)
	}
	sourceState.To.Directories = directories
	return writeSkillToMetadata(state.FromPath, sourceState.To)
}

func removeDirectoryFromSourceRelation(sourcePath, directoryPath string) error {
	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	if sourcePath == "" {
		return nil
	}
	state, err := readSkillRelationState(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !state.HasTo {
		return nil
	}
	filtered := make([]string, 0, len(state.To.Directories))
	for _, directory := range state.To.Directories {
		if filepath.Clean(directory) == filepath.Clean(directoryPath) {
			continue
		}
		filtered = append(filtered, directory)
	}
	state.To.Directories = filtered
	return writeSkillToMetadata(sourcePath, state.To)
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func uniqueSortedPatterns(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := filepath.ToSlash(strings.TrimSpace(value))
		trimmed = strings.TrimPrefix(trimmed, "./")
		if trimmed == "" || trimmed == "." {
			continue
		}
		trimmed = path.Clean(trimmed)
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func copyFile(sourcePath, targetPath string, mode os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() { _ = targetFile.Close() }()

	_, err = io.Copy(targetFile, sourceFile)
	return err
}

func listFileNodes(rootPath, currentPath string) ([]FileNode, error) {
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, err
	}
	nodes := make([]FileNode, 0, len(entries))
	for _, entry := range entries {
		if isIgnoredName(entry.Name()) {
			continue
		}
		fullPath := filepath.Join(currentPath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		relPath, err := filepath.Rel(rootPath, fullPath)
		if err != nil {
			return nil, err
		}
		modifiedAt := info.ModTime()
		node := FileNode{
			Name:       entry.Name(),
			Path:       filepath.ToSlash(relPath),
			IsDir:      entry.IsDir(),
			ModifiedAt: &modifiedAt,
		}
		if entry.IsDir() {
			children, err := listFileNodes(rootPath, fullPath)
			if err != nil {
				return nil, err
			}
			node.Children = children
		} else {
			node.Size = info.Size()
		}
		if node.Path == "." {
			node.Path = ""
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir == nodes[j].IsDir {
			return nodes[i].Name < nodes[j].Name
		}
		return nodes[i].IsDir
	})
	return nodes, nil
}
