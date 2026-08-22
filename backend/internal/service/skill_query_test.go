package service

import (
	"backend-go/internal/models"
	"context"
	"path/filepath"
	"testing"
)

func TestFuzzyMatchRank(t *testing.T) {
	cases := []struct {
		name  string
		token string
		value string
		want  int
	}{
		{"exact", "jira", "jira", fuzzyRankExact},
		{"exact case-insensitive", "Jira", "jira", fuzzyRankExact},
		{"prefix", "jira", "jira-sync", fuzzyRankPrefix},
		{"substring", "jira", "my-jira-tool", fuzzyRankSubstring},
		{"letter-order is not a hit", "jbr", "jira-browser", fuzzyRankNone},
		{"no match", "jira", "github", fuzzyRankNone},
		{"chinese prefix", "报销", "报销助手", fuzzyRankPrefix},
		{"chinese substring", "助手", "报销助手", fuzzyRankSubstring},
		{"empty token", "", "jira", fuzzyRankNone},
		{"empty value", "jira", "", fuzzyRankNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fuzzyMatchRank(tc.token, tc.value); got != tc.want {
				t.Fatalf("fuzzyMatchRank(%q, %q) = %d, want %d", tc.token, tc.value, got, tc.want)
			}
		})
	}
}

func TestSkillQueryScore(t *testing.T) {
	skill := models.Skill{
		Name: "jira-sync",
		Slug: "jira-sync",
		Provider: models.Provider{
			Name: "Workspace Skills",
		},
		Tags:    []string{"issue-tracking"},
		Summary: "Sync issues from Jira",
	}

	cases := []struct {
		name    string
		tokens  []string
		wantPos bool
	}{
		{"single exact-ish token", []string{"jira-sync"}, true},
		{"token in summary", []string{"issues"}, true},
		{"token in tag", []string{"tracking"}, true},
		{"provider name is not a written field", []string{"workspace"}, false},
		{"multi token AND", []string{"jira", "sync"}, true},
		{"one unmatched token rejects", []string{"jira", "github"}, false},
		{"no match", []string{"kubernetes"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := skillQueryScore(tc.tokens, skill)
			if tc.wantPos && got <= 0 {
				t.Fatalf("expected positive score, got %d", got)
			}
			if !tc.wantPos && got != 0 {
				t.Fatalf("expected zero score, got %d", got)
			}
		})
	}
}

func TestSkillQueryScoreRanksNameMatchesHighest(t *testing.T) {
	nameSkill := models.Skill{Name: "jira"}
	summarySkill := models.Skill{Name: "unrelated", Summary: "mentions jira somewhere"}

	nameScore := skillQueryScore([]string{"jira"}, nameSkill)
	summaryScore := skillQueryScore([]string{"jira"}, summarySkill)
	if nameScore <= summaryScore {
		t.Fatalf("expected name match (%d) to outscore summary match (%d)", nameScore, summaryScore)
	}
}

func TestListSkillsQueryKeepsOnlyWrittenContiguousHits(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	provider := createTestProvider(t, db, "Workspace Skills", filepath.Join(baseDir, "provider"))

	createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "jira"), "jira")
	createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "jira-sync"), "jira-sync")
	createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "my-jira-helper"), "my-jira-helper")
	createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "job-iteration-report-app"), "job-iteration-report-app")
	createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "github-actions"), "github-actions")

	skills, err := service.ListSkills(ctx, SkillListFilters{Query: "jira", Sort: "name"})
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}

	var names []string
	for _, skill := range skills {
		names = append(names, skill.Name)
	}
	want := []string{"jira", "jira-sync", "my-jira-helper"}
	if len(names) != len(want) {
		t.Fatalf("expected %d matches, got %d: %v", len(want), len(names), names)
	}
	for index, name := range want {
		if names[index] != name {
			t.Fatalf("expected order %v, got %v", want, names)
		}
	}
}

func TestListSkillsQueryRejectsLetterOrderAndProviderName(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	provider := createTestProvider(t, db, "Auth Platform", filepath.Join(baseDir, "provider"))

	createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "jira-browser"), "jira-browser")
	providerOnly := createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "notes"), "notes")
	providerOnly.Summary = "internal notes"
	if err := db.Save(providerOnly).Error; err != nil {
		t.Fatalf("update summary: %v", err)
	}

	letterOrder, err := service.ListSkills(ctx, SkillListFilters{Query: "jbr"})
	if err != nil {
		t.Fatalf("ListSkills jbr: %v", err)
	}
	if len(letterOrder) != 0 {
		t.Fatalf("expected no letter-order hits, got %d", len(letterOrder))
	}

	byProviderName, err := service.ListSkills(ctx, SkillListFilters{Query: "platform"})
	if err != nil {
		t.Fatalf("ListSkills provider name: %v", err)
	}
	if len(byProviderName) != 0 {
		t.Fatalf("expected provider name not to match, got %d", len(byProviderName))
	}

	unknown, err := service.ListSkills(ctx, SkillListFilters{Query: "zzzz-not-a-skill"})
	if err != nil {
		t.Fatalf("ListSkills unknown: %v", err)
	}
	if len(unknown) != 0 {
		t.Fatalf("expected empty unknown query, got %d", len(unknown))
	}
}

func TestListSkillsQueryMatchesTagsAliasesAndAbbreviations(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	provider := createTestProvider(t, db, "Workspace Skills", filepath.Join(baseDir, "provider"))

	pdl := createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "product-development-lifecycle"), "product-development-lifecycle")
	pdl.Tags = []string{"pdl"}
	if err := db.Save(pdl).Error; err != nil {
		t.Fatalf("update tags: %v", err)
	}

	guidelines := createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "karpathy-guidelines"), "karpathy-guidelines")
	guidelines.Summary = "style guide for agents"
	if err := db.Save(guidelines).Error; err != nil {
		t.Fatalf("update guidelines: %v", err)
	}

	createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "browserauth"), "browserauth")
	authTagged := createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "gate"), "gate")
	authTagged.Tags = []string{"auth"}
	if err := db.Save(authTagged).Error; err != nil {
		t.Fatalf("update auth tag: %v", err)
	}
	named := createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "login"), "login")
	named.Summary = "authentication helpers"
	if err := db.Save(named).Error; err != nil {
		t.Fatalf("update auth summary: %v", err)
	}

	pdlHits, err := service.ListSkills(ctx, SkillListFilters{Query: "PDL"})
	if err != nil {
		t.Fatalf("ListSkills PDL: %v", err)
	}
	if len(pdlHits) != 1 || pdlHits[0].Name != "product-development-lifecycle" {
		names := make([]string, 0, len(pdlHits))
		for _, skill := range pdlHits {
			names = append(names, skill.Name)
		}
		t.Fatalf("expected only product-development-lifecycle, got %v", names)
	}

	authHits, err := service.ListSkills(ctx, SkillListFilters{Query: "auth", Sort: "name"})
	if err != nil {
		t.Fatalf("ListSkills auth: %v", err)
	}
	got := map[string]bool{}
	for _, skill := range authHits {
		got[skill.Name] = true
	}
	for _, name := range []string{"browserauth", "gate", "login"} {
		if !got[name] {
			t.Fatalf("expected %s among auth hits, got %v", name, got)
		}
	}
	if got["karpathy-guidelines"] || got["product-development-lifecycle"] {
		t.Fatalf("unexpected auth hits: %v", got)
	}
}

func TestListSkillsFuzzyQueryMatchesTagsAndChineseSummary(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	provider := createTestProvider(t, db, "Workspace Skills", filepath.Join(baseDir, "provider"))

	tagSkill := createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "expense-flow"), "expense-flow")
	tagSkill.Tags = []string{"报销", "费用"}
	if err := db.Save(tagSkill).Error; err != nil {
		t.Fatalf("update tags: %v", err)
	}

	summarySkill := createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "reimbursement"), "reimbursement")
	summarySkill.Summary = "报销规则讲解与流程可视化"
	if err := db.Save(summarySkill).Error; err != nil {
		t.Fatalf("update summary: %v", err)
	}

	createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "other"), "other")

	skills, err := service.ListSkills(ctx, SkillListFilters{Query: "报销"})
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(skills) != 2 {
		names := make([]string, 0, len(skills))
		for _, skill := range skills {
			names = append(names, skill.Name)
		}
		t.Fatalf("expected 2 matches, got %v", names)
	}
	if skills[0].Name != "expense-flow" || skills[1].Name != "reimbursement" {
		t.Fatalf("expected [expense-flow reimbursement], got [%s %s]", skills[0].Name, skills[1].Name)
	}
}

func TestListSkillsQueryCombinesWithProviderAndStatusFilters(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	first := createTestProvider(t, db, "First", filepath.Join(baseDir, "first"))
	second := createTestProvider(t, db, "Second", filepath.Join(baseDir, "second"))

	createTestSkill(t, db, first, filepath.Join(baseDir, "first", "jira"), "jira")
	other := createTestSkill(t, db, second, filepath.Join(baseDir, "second", "jira"), "jira")
	other.Status = "invalid"
	if err := db.Save(other).Error; err != nil {
		t.Fatalf("update status: %v", err)
	}
	createTestSkill(t, db, second, filepath.Join(baseDir, "second", "github"), "github")

	byProvider, err := service.ListSkills(ctx, SkillListFilters{Query: "jira", Provider: second.Zid})
	if err != nil {
		t.Fatalf("ListSkills with provider: %v", err)
	}
	if len(byProvider) != 1 || byProvider[0].Provider.Zid != second.Zid {
		t.Fatalf("expected one skill from Second, got %d", len(byProvider))
	}

	byStatus, err := service.ListSkills(ctx, SkillListFilters{Query: "jira", Status: "invalid"})
	if err != nil {
		t.Fatalf("ListSkills with status: %v", err)
	}
	if len(byStatus) != 1 || byStatus[0].Status != "invalid" {
		t.Fatalf("expected one invalid skill, got %d", len(byStatus))
	}

	noMatch, err := service.ListSkills(ctx, SkillListFilters{Query: "jira github"})
	if err != nil {
		t.Fatalf("ListSkills multi-token: %v", err)
	}
	if len(noMatch) != 0 {
		t.Fatalf("expected no skill to match both tokens, got %d", len(noMatch))
	}

	andHits, err := service.ListSkills(ctx, SkillListFilters{Query: "product development"})
	if err != nil {
		t.Fatalf("ListSkills AND empty: %v", err)
	}
	if len(andHits) != 0 {
		t.Fatalf("expected no AND hits in this fixture, got %d", len(andHits))
	}
}

func TestListSkillsQueryRequiresEveryKeyword(t *testing.T) {
	db := openCatalogTestDB(t)
	service := NewCatalogService(db)
	ctx := context.Background()

	baseDir := t.TempDir()
	provider := createTestProvider(t, db, "Workspace Skills", filepath.Join(baseDir, "provider"))

	both := createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "product-development-lifecycle"), "product-development-lifecycle")
	both.Summary = "product development process"
	if err := db.Save(both).Error; err != nil {
		t.Fatalf("update both: %v", err)
	}
	onlyProduct := createTestSkill(t, db, provider, filepath.Join(baseDir, "provider", "product-ops"), "product-ops")
	onlyProduct.Summary = "product operations"
	if err := db.Save(onlyProduct).Error; err != nil {
		t.Fatalf("update product-only: %v", err)
	}

	skills, err := service.ListSkills(ctx, SkillListFilters{Query: "product development"})
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(skills) != 1 || skills[0].Name != "product-development-lifecycle" {
		names := make([]string, 0, len(skills))
		for _, skill := range skills {
			names = append(names, skill.Name)
		}
		t.Fatalf("expected only product-development-lifecycle, got %v", names)
	}
}
