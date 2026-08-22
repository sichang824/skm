package main

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"backend-go/internal/models"
)

// digestSummaryLimit bounds the per-skill summary in digest output so the
// whole catalog stays cheap to inject into an LLM context.
const digestSummaryLimit = 44

// digestCommandLimit bounds how many command names render per skill; the
// rest collapse to "…+N" (full detail stays in `skills get <zid> --commands`).
const digestCommandLimit = 8

// digestEntry is one rendered digest row after collapsing same-name rows.
type digestEntry struct {
	skill models.Skill
	dupes int // rows sharing this name, including the canonical one
}

// renderSkillsDigest writes a compact one-line-per-skill catalog digest
// intended for direct injection into an LLM context (the skm skill embeds it
// via dynamic context so agents can locate skills without extra queries).
// Each skill is one tab-separated line:
//
//	NAME  ZID  PROVIDER  STATUS  COMMANDS  SUMMARY
//
// Same-name rows (multi-version Shiju directories, source + attached copies)
// collapse into one canonical row suffixed "×N": preference order is source
// over copy, ready over non-ready, then newest modification. STATUS is empty
// for ready skills; "(copy)" marks a canonical row that is itself a copy
// (its source is absent from the catalog). COMMANDS is "-" when nothing is
// declared; otherwise space-separated names, "*" = confirm required (--yes),
// truncated to digestCommandLimit. Conflict losers (isEffective == false)
// are omitted entirely. The db DSN is printed in the header so a
// dev-database CWD mix-up is visible at a glance.
//
// The whole output targets well under 30KB: larger injections are truncated
// when embedded as skill dynamic context.
func renderSkillsDigest(out io.Writer, skills []models.Skill, dbDSN string) {
	effective := make([]models.Skill, 0, len(skills))
	omitted := 0
	for _, skill := range skills {
		if !skill.IsEffective {
			omitted++
			continue
		}
		effective = append(effective, skill)
	}

	entries, folded := collapseDigestRows(effective)
	executable := 0
	for _, entry := range entries {
		if len(entry.skill.Commands) > 0 {
			executable++
		}
	}

	fmt.Fprintf(out, "# skm catalog digest: %d skills (%d executable, %d duplicates folded, %d conflict-losers omitted) | db: %s\n",
		len(entries), executable, folded, omitted, dbDSN)
	fmt.Fprintln(out, "NAME\tZID\tPROVIDER\tSTATUS\tCOMMANDS\tSUMMARY")
	for _, entry := range entries {
		skill := entry.skill
		name := skill.Name
		if entry.dupes > 1 {
			name = fmt.Sprintf("%s ×%d", name, entry.dupes)
		}
		status := string(skill.Status)
		if status == "ready" {
			status = ""
		}
		if isDigestCopy(skill) {
			status += "(copy)"
		}
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\n",
			name,
			skill.Zid,
			skill.Provider.Name,
			status,
			digestCommands(skill.Commands),
			digestSummary(skill.Summary),
		)
	}
}

// collapseDigestRows folds same-name rows into one canonical entry per name,
// preserving first-appearance order. It returns the entries and the number
// of non-canonical rows folded away.
func collapseDigestRows(skills []models.Skill) ([]digestEntry, int) {
	index := make(map[string]int, len(skills))
	entries := make([]digestEntry, 0, len(skills))
	folded := 0
	for _, skill := range skills {
		i, ok := index[skill.Name]
		if !ok {
			index[skill.Name] = len(entries)
			entries = append(entries, digestEntry{skill: skill, dupes: 1})
			continue
		}
		folded++
		entries[i].dupes++
		if digestPreferred(skill, entries[i].skill) {
			entries[i].skill = skill
		}
	}
	return entries, folded
}

// digestPreferred reports whether a is a better canonical row than b:
// a source beats a copy, ready beats non-ready, then the newest modification
// wins (for versioned directories that is the latest version).
func digestPreferred(a, b models.Skill) bool {
	aCopy, bCopy := isDigestCopy(a), isDigestCopy(b)
	if aCopy != bCopy {
		return !aCopy
	}
	aReady := a.Status == "ready"
	bReady := b.Status == "ready"
	if aReady != bReady {
		return aReady
	}
	return digestModified(a).After(digestModified(b))
}

func digestModified(skill models.Skill) time.Time {
	if skill.LastModifiedAt != nil {
		return *skill.LastModifiedAt
	}
	return skill.LastScannedAt
}

func isDigestCopy(skill models.Skill) bool {
	return skill.Relation != nil && skill.Relation.Mode == "from"
}

func digestCommands(commands []models.SkillCommand) string {
	if len(commands) == 0 {
		return "-"
	}
	names := make([]string, 0, len(commands))
	for _, command := range commands {
		name := command.Name
		if command.Confirm != "" {
			name += "*"
		}
		names = append(names, name)
	}
	if len(names) > digestCommandLimit {
		extra := len(names) - digestCommandLimit
		names = append(names[:digestCommandLimit], fmt.Sprintf("…+%d", extra))
	}
	return strings.Join(names, " ")
}

// digestSummary collapses a skill summary to a single line of at most
// digestSummaryLimit runes. A bare "---" is a frontmatter-parse artifact and
// renders as empty.
func digestSummary(summary string) string {
	text := strings.Join(strings.Fields(summary), " ")
	text = strings.TrimPrefix(text, "---")
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > digestSummaryLimit {
		runes := []rune(text)
		text = string(runes[:digestSummaryLimit]) + "…"
	}
	return text
}
