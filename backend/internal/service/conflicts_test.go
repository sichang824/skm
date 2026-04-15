package service

import (
	"backend-go/internal/models"
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
