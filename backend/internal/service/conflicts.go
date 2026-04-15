package service

import (
	"backend-go/internal/models"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type ConflictGroup struct {
	Kind              string         `json:"kind"`
	Key               string         `json:"key"`
	EffectiveSkillZid string         `json:"effectiveSkillZid,omitempty"`
	Skills            []models.Skill `json:"skills"`
}

func buildConflictGroups(skills []models.Skill) []ConflictGroup {
	groups := make([]ConflictGroup, 0)
	byName := make(map[string][]models.Skill)
	byPath := make(map[string][]models.Skill)

	for _, skill := range skills {
		byName[strings.ToLower(skill.Name)] = append(byName[strings.ToLower(skill.Name)], skill)
		byPath[skill.RootPath] = append(byPath[skill.RootPath], skill)
	}

	for key, groupSkills := range byName {
		if len(groupSkills) < 2 {
			continue
		}
		sortSkillsByPriority(groupSkills)
		groups = append(groups, ConflictGroup{
			Kind:              classifyNameConflict(groupSkills),
			Key:               key,
			EffectiveSkillZid: groupSkills[0].Zid,
			Skills:            groupSkills,
		})
	}

	for key, groupSkills := range byPath {
		if len(groupSkills) < 2 {
			continue
		}
		sortSkillsByPriority(groupSkills)
		groups = append(groups, ConflictGroup{
			Kind:              "path_duplicate",
			Key:               key,
			EffectiveSkillZid: groupSkills[0].Zid,
			Skills:            groupSkills,
		})
	}

	return groups
}

func classifyNameConflict(skills []models.Skill) string {
	hashes := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		hashes[skill.ContentHash] = struct{}{}
	}
	if len(hashes) > 1 {
		return "name_content_diff"
	}
	return "name_duplicate"
}

func sortSkillsByPriority(skills []models.Skill) {
	sort.Slice(skills, func(i, j int) bool {
		left := skills[i]
		right := skills[j]
		if left.Provider.Priority != right.Provider.Priority {
			return left.Provider.Priority > right.Provider.Priority
		}
		if !left.LastScannedAt.Equal(right.LastScannedAt) {
			return left.LastScannedAt.After(right.LastScannedAt)
		}
		return left.Zid < right.Zid
	})
}

func rebuildConflictState(tx *gorm.DB) (int, error) {
	var skills []models.Skill
	if err := tx.
		Select(
			"skills.id",
			"skills.zid",
			"skills.provider_id",
			"skills.name",
			"skills.root_path",
			"skills.content_hash",
			"skills.last_scanned_at",
			"skills.is_conflict",
			"skills.is_effective",
		).
		Preload("Provider").
		Joins("JOIN providers ON providers.id = skills.provider_id").
		Find(&skills).Error; err != nil {
		return 0, err
	}

	groups := buildConflictGroups(filterEnabledSkills(skills))
	state := make(map[uint]models.Skill)
	for _, skill := range skills {
		skill.IsConflict = false
		skill.IsEffective = skill.Provider.Enabled
		skill.ConflictKinds = []string{}
		state[skill.ID] = skill
	}

	for _, group := range groups {
		for index, skill := range group.Skills {
			current := state[skill.ID]
			current.IsConflict = true
			current.ConflictKinds = appendUnique(current.ConflictKinds, group.Kind)
			if index > 0 {
				current.IsEffective = false
			}
			state[skill.ID] = current
		}
	}

	for _, skill := range state {
		conflictKindsJSON, err := marshalStringSliceJSON(skill.ConflictKinds)
		if err != nil {
			return 0, fmt.Errorf("marshal conflict kinds for skill %s: %w", skill.Zid, err)
		}
		if err := tx.Model(&models.Skill{}).
			Where("id = ?", skill.ID).
			Updates(map[string]any{
				"is_conflict":    skill.IsConflict,
				"is_effective":   skill.IsEffective,
				"conflict_kinds": conflictKindsJSON,
			}).Error; err != nil {
			return 0, err
		}
	}

	return len(groups), nil
}

func marshalStringSliceJSON(values []string) (string, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func filterEnabledSkills(skills []models.Skill) []models.Skill {
	filtered := make([]models.Skill, 0, len(skills))
	for _, skill := range skills {
		if skill.Provider.Enabled {
			filtered = append(filtered, skill)
		}
	}
	return filtered
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
