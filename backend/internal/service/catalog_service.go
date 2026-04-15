package service

import (
	"backend-go/internal/models"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrSkillNotFound    = errors.New("skill not found")
	ErrScanJobNotFound  = errors.New("scan job not found")
	ErrInvalidInput     = errors.New("invalid input")
	ErrBinaryFile       = errors.New("binary file preview is not supported")
)

type CatalogService struct {
	db *gorm.DB
}

type ProviderInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
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
