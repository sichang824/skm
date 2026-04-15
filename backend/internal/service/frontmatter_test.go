package service

import "testing"

func TestParseSkillDocumentExtractsFrontmatter(t *testing.T) {
	doc := `---
name: Sample Skill
category: tooling
summary: concise summary
tags:
  - search
  - scan
---

# Heading

Body paragraph.`

	parsed, err := parseSkillDocument(doc)
	if err != nil {
		t.Fatalf("parseSkillDocument returned error: %v", err)
	}
	if parsed.Name != "Sample Skill" {
		t.Fatalf("expected name Sample Skill, got %q", parsed.Name)
	}
	if parsed.Category != "tooling" {
		t.Fatalf("expected category tooling, got %q", parsed.Category)
	}
	if parsed.Summary != "concise summary" {
		t.Fatalf("expected summary from frontmatter, got %q", parsed.Summary)
	}
	if len(parsed.Tags) != 2 || parsed.Tags[0] != "search" || parsed.Tags[1] != "scan" {
		t.Fatalf("unexpected tags: %#v", parsed.Tags)
	}
	if parsed.Hash == "" {
		t.Fatal("expected content hash to be populated")
	}
}
