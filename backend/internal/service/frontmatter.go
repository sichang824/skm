package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var frontmatterPattern = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?(.*)$`)

type ParsedSkillDocument struct {
	Frontmatter map[string]any
	Body        string
	Summary     string
	Category    string
	Tags        []string
	Name        string
	Hash        string
}

func parseSkillDocument(content string) (*ParsedSkillDocument, error) {
	result := &ParsedSkillDocument{
		Frontmatter: map[string]any{},
		Body:        content,
		Hash:        hashContent(content),
	}

	if matches := frontmatterPattern.FindStringSubmatch(content); len(matches) == 3 {
		if err := yaml.Unmarshal([]byte(matches[1]), &result.Frontmatter); err != nil {
			return nil, fmt.Errorf("frontmatter parse failed: %w", err)
		}
		result.Body = strings.TrimSpace(matches[2])
	}

	result.Name = firstString(result.Frontmatter, "name")
	result.Category = firstString(result.Frontmatter, "category")
	result.Summary = firstString(result.Frontmatter, "summary", "description")
	if result.Summary == "" {
		result.Summary = summarizeBody(result.Body)
	}
	result.Tags = readTags(result.Frontmatter["tags"])

	return result, nil
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func summarizeBody(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func readTags(value any) []string {
	if value == nil {
		return []string{}
	}
	if text, ok := value.(string); ok {
		parts := strings.Split(text, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	items, ok := value.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			trimmed := strings.TrimSpace(text)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

func slugify(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer("_", "-", " ", "-", "/", "-", "\\", "-")
	value = replacer.Replace(value)
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return !(r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	return strings.Join(parts, "-")
}
