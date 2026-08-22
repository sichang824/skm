package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"backend-go/internal/models"
)

func TestDigestSummary(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "Manage Apple Notes via memo CLI.", "Manage Apple Notes via memo CLI."},
		{"collapses whitespace", "line one\r\n\nline two\t\ttab", "line one line two tab"},
		{"frontmatter artifact", "---", ""},
		{"artifact with trailing text", "---\r\nreal summary", "real summary"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := digestSummary(tc.input); got != tc.want {
				t.Fatalf("digestSummary(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestDigestSummaryTruncatesLongText(t *testing.T) {
	input := strings.Repeat("技", digestSummaryLimit+10)
	got := digestSummary(input)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis suffix, got %q", got)
	}
	runes := []rune(got)
	if len(runes) != digestSummaryLimit+1 {
		t.Fatalf("expected %d runes plus ellipsis, got %d", digestSummaryLimit, len(runes))
	}
}

func TestDigestCommands(t *testing.T) {
	if got := digestCommands(nil); got != "-" {
		t.Fatalf("no commands should render '-', got %q", got)
	}

	commands := []models.SkillCommand{
		{Name: "list"},
		{Name: "delete", Confirm: "always"},
		{Name: "push", Env: []string{"API_KEY"}},
	}
	if got, want := digestCommands(commands), "list delete* push"; got != want {
		t.Fatalf("digestCommands = %q, want %q", got, want)
	}

	many := make([]models.SkillCommand, digestCommandLimit+3)
	for i := range many {
		many[i] = models.SkillCommand{Name: "cmd" + string(rune('a'+i))}
	}
	got := digestCommands(many)
	if !strings.HasSuffix(got, " …+3") {
		t.Fatalf("expected overflow suffix '…+3', got %q", got)
	}
	if parts := strings.Split(got, " "); len(parts) != digestCommandLimit+1 {
		t.Fatalf("expected %d rendered parts, got %d (%q)", digestCommandLimit+1, len(parts), got)
	}
}

func TestRenderSkillsDigest(t *testing.T) {
	skills := []models.Skill{
		{
			Name:     "apple-notes",
			Status:   "ready",
			Summary:  "Manage Apple Notes.",
			Provider: models.Provider{Name: "Hermes"},
		},
		{
			Name:     "fireworks",
			Status:   "ready",
			Summary:  "Demo skill.",
			Provider: models.Provider{Name: "Workspace Skills"},
			Commands: []models.SkillCommand{
				{Name: "build"},
				{Name: "publish", Confirm: "always"},
			},
		},
		{
			Name:     "skm",
			Status:   "ready",
			Summary:  "The skm skill.",
			Provider: models.Provider{Name: "Claude Skills"},
			Relation: &models.SkillRelation{Mode: "from", FromPath: "/src/skm"},
		},
		{
			Name:        "old-version-loser",
			Status:      "ready",
			Summary:     "Superseded by a conflict winner.",
			Provider:    models.Provider{Name: "Shiju Skills"},
			IsConflict:  true,
			IsEffective: false,
		},
	}
	skills[0].Zid = "ZIDNOTES00000000"
	skills[1].Zid = "ZIDFIRE000000000"
	skills[2].Zid = "ZIDSKM0000000000"
	skills[3].Zid = "ZIDLOSER00000000"
	// Effective rows: IsEffective defaults to false in fixtures; mark them.
	skills[0].IsEffective = true
	skills[1].IsEffective = true
	skills[2].IsEffective = true

	var buf bytes.Buffer
	renderSkillsDigest(&buf, skills, "/Users/x/.skm/app.db")
	out := buf.String()

	for _, want := range []string{
		"# skm catalog digest: 3 skills (1 executable, 0 duplicates folded, 1 conflict-losers omitted) | db: /Users/x/.skm/app.db",
		"NAME\tZID\tPROVIDER\tSTATUS\tCOMMANDS\tSUMMARY",
		"apple-notes\tZIDNOTES00000000\tHermes\t\t-\tManage Apple Notes.",
		"fireworks\tZIDFIRE000000000\tWorkspace Skills\t\tbuild publish*\tDemo skill.",
		"skm\tZIDSKM0000000000\tClaude Skills\t(copy)\t-\tThe skm skill.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("digest output missing %q\n--- output ---\n%s", want, out)
		}
	}
	if strings.Contains(out, "old-version-loser") {
		t.Errorf("conflict loser should be omitted\n--- output ---\n%s", out)
	}
	if got := len(strings.Split(strings.TrimRight(out, "\n"), "\n")); got != 5 {
		t.Errorf("expected 5 lines (1 header comment + 1 column header + 3 skills), got %d", got)
	}
}

func TestCollapseDigestRows(t *testing.T) {
	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	shijuOld := models.Skill{Name: "shuju-knowledge-base", Status: "invalid", LastModifiedAt: &older}
	shijuOld.Zid = "OLDDDDDDDDDDDDDD"
	shijuOld.Provider = models.Provider{Name: "Shiju Skills"}
	shijuNew := models.Skill{Name: "shuju-knowledge-base", Status: "invalid", LastModifiedAt: &newer}
	shijuNew.Zid = "NEWDDDDDDDDDDDDD"
	shijuNew.Provider = models.Provider{Name: "Shiju Skills"}

	source := models.Skill{Name: "skm", Status: "ready"}
	source.Zid = "SOURCESKMSKMSKM0"
	source.Provider = models.Provider{Name: "Workspace Skills"}
	copyRow := models.Skill{Name: "skm", Status: "ready",
		Relation: &models.SkillRelation{Mode: "from", FromPath: "/src/skm"}}
	copyRow.Zid = "COPYSKMSKMSKMSKM"
	copyRow.Provider = models.Provider{Name: "Claude Skills"}

	// Feed the copy first to prove the source still wins as canonical.
	entries, folded := collapseDigestRows([]models.Skill{copyRow, source, shijuOld, shijuNew})
	if folded != 2 {
		t.Fatalf("folded = %d, want 2", folded)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}

	first := entries[0]
	if first.skill.Zid != "SOURCESKMSKMSKM0" || first.dupes != 2 {
		t.Fatalf("skm canonical should be the source with dupes=2, got zid=%s dupes=%d", first.skill.Zid, first.dupes)
	}
	second := entries[1]
	if second.skill.Zid != "NEWDDDDDDDDDDDDD" || second.dupes != 2 {
		t.Fatalf("shuju canonical should be the newest version with dupes=2, got zid=%s dupes=%d", second.skill.Zid, second.dupes)
	}
}

func TestRenderSkillsDigestFoldsSameName(t *testing.T) {
	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := models.Skill{Name: "kb", Status: "invalid", Summary: "KB access.", LastModifiedAt: &older, IsEffective: true}
	a.Zid = "ZIDKB00000000001"
	a.Provider = models.Provider{Name: "Shiju Skills"}
	b := models.Skill{Name: "kb", Status: "invalid", Summary: "KB access.", LastModifiedAt: &older, IsEffective: true}
	b.Zid = "ZIDKB00000000002"
	b.Provider = models.Provider{Name: "Shiju Skills"}

	var buf bytes.Buffer
	renderSkillsDigest(&buf, []models.Skill{a, b}, "/db")
	out := buf.String()
	if !strings.Contains(out, "kb ×2\tZIDKB00000000001\tShiju Skills\tinvalid") {
		t.Errorf("expected folded row with ×2 marker\n--- output ---\n%s", out)
	}
	if !strings.Contains(out, "1 duplicates folded") {
		t.Errorf("header should report the folded count\n--- output ---\n%s", out)
	}
}
