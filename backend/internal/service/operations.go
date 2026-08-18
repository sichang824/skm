package service

import (
	"backend-go/internal/models"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Operations section generation (design doc §8): the manifest is the single
// source of truth for executable commands, and the SKILL.md Operations
// section is generated from it so the prose an agent reads can never drift
// from what `skills exec` actually runs.
const (
	// OperationsHeading is the section heading managed by the generator.
	OperationsHeading = "## Operations"
	// operationsBeginMark / operationsEndMark delimit generated content so
	// regeneration replaces exactly what was generated before.
	operationsBeginMark = "<!-- skm:operations:begin — generated from package.json; do not edit -->"
	operationsEndMark   = "<!-- skm:operations:end -->"
)

// OperationsResult describes one generation run.
type OperationsResult struct {
	SkillZid  string `json:"skillZid,omitempty"`
	SkillName string `json:"skillName,omitempty"`
	SkillRoot string `json:"skillRoot"`
	// Changed reports whether SKILL.md content differs from before.
	Changed bool `json:"changed"`
	// Written reports whether the file was actually written (false in
	// --check mode).
	Written bool `json:"written"`
	// CommandCount is the number of rendered commands.
	CommandCount int `json:"commandCount"`
}

// SkillOperationsRoot resolves the directory whose SKILL.md should carry the
// Operations section: linked copies resolve to their source so generated
// docs live at the single source of truth.
func SkillOperationsRoot(skill *models.Skill) string {
	return resolveExecDir(skill)
}

// GenerateOperations renders the Operations section from the skill's
// package.json and merges it into SKILL.md. With checkOnly it reports
// whether the file would change without writing.
func GenerateOperations(skillRoot, skillZid, skillName string, checkOnly bool) (*OperationsResult, error) {
	manifest, err := LoadManifest(skillRoot)
	if err != nil {
		return nil, err
	}

	skillMdPath := filepath.Join(skillRoot, "SKILL.md")
	data, err := os.ReadFile(skillMdPath)
	if err != nil {
		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}

	section, commandCount := RenderOperationsSection(skillZid, manifest)
	merged := replaceOperationsSection(string(data), section)

	result := &OperationsResult{
		SkillZid:     skillZid,
		SkillName:    skillName,
		SkillRoot:    skillRoot,
		Changed:      merged != string(data),
		CommandCount: commandCount,
	}
	if checkOnly || !result.Changed {
		return result, nil
	}
	if err := os.WriteFile(skillMdPath, []byte(merged), 0o644); err != nil {
		return nil, fmt.Errorf("write SKILL.md: %w", err)
	}
	result.Written = true
	return result, nil
}

// RenderOperationsSection renders the full Operations section (heading
// included) for one manifest. zid may be empty, in which case the
// `<skill-zid>` placeholder is kept. The second return value is the number
// of rendered commands.
func RenderOperationsSection(zid string, manifest *ParsedManifest) (string, int) {
	target := "<skill-zid>"
	if strings.TrimSpace(zid) != "" {
		target = strings.TrimSpace(zid)
	}

	var builder strings.Builder
	builder.WriteString(OperationsHeading + "\n\n")
	builder.WriteString(operationsBeginMark + "\n\n")
	builder.WriteString("Executable commands declared in package.json. Run them through skm exec:\n")
	builder.WriteString("only declared commands can run, and skm enforces env, input, and confirm\n")
	builder.WriteString("pre-checks before starting anything.\n")

	if len(manifest.RuntimeEnv) > 0 {
		fmt.Fprintf(&builder, "\nRequired environment: %s — inject with `--env KEY=VAL`.\n", backtickList(manifest.RuntimeEnv))
	}
	if manifest.SetupCommand != "" {
		fmt.Fprintf(&builder, "\nFirst use runs setup `%s` automatically (idempotent; maintained by skm).\n", manifest.SetupCommand)
	}

	count := 0
	for _, command := range manifest.Commands {
		if command.Name == manifest.SetupCommand {
			// Setup is preparation infrastructure, not an operation the
			// agent invokes directly; exec runs it automatically.
			continue
		}
		count++
		fmt.Fprintf(&builder, "\n### %s\n\n", command.Name)
		if command.Description != "" {
			builder.WriteString(command.Description + "\n\n")
		}
		if command.Confirm != "" {
			fmt.Fprintf(&builder, "> ⚠️ Requires confirmation: %s — pass `--yes` once the user agrees.\n\n", command.Confirm)
		}
		if len(command.Env) > 0 {
			fmt.Fprintf(&builder, "Required environment for this command: %s.\n\n", backtickList(command.Env))
		}
		if command.InputVia != "" {
			fmt.Fprintf(&builder, "Input: structured JSON delivered via %s; validated against the declared schema before running.\n\n", command.InputVia)
		}
		fmt.Fprintf(&builder, "Command: `%s`\n", operationCommandTemplate(target, &command))
	}

	builder.WriteString("\n" + operationsEndMark)
	return builder.String(), count
}

func operationCommandTemplate(target string, command *models.SkillCommand) string {
	template := fmt.Sprintf("skm skills exec %s %s", target, command.Name)
	if command.InputVia != "" {
		template += " --input '<json>'"
	} else {
		template += " -- <args...>"
	}
	return template
}

func backtickList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "`"+value+"`")
	}
	return strings.Join(quoted, ", ")
}

var topLevelHeadingPattern = regexp.MustCompile(`^#{1,2} `)

// replaceOperationsSection merges a rendered section into a SKILL.md
// document. It replaces, in order of preference: a previously generated
// block (identified by its markers, plus the heading directly above it), an
// existing unmarked `## Operations` section up to the next level-1/2
// heading; otherwise it appends the section at the end of the document.
func replaceOperationsSection(document, section string) string {
	hadTrailingNewline := strings.HasSuffix(document, "\n")
	lines := strings.Split(strings.TrimRight(document, "\n"), "\n")
	sectionLines := strings.Split(section, "\n")

	start, end, found := locateOperationsBlock(lines)
	var prefix, suffix []string
	if found {
		prefix = trimTrailingBlank(lines[:start])
		suffix = trimLeadingBlank(lines[end:])
	} else {
		prefix = trimTrailingBlank(lines)
	}

	groups := make([][]string, 0, 3)
	if len(prefix) > 0 {
		groups = append(groups, prefix)
	}
	groups = append(groups, sectionLines)
	if len(suffix) > 0 {
		groups = append(groups, suffix)
	}

	out := make([]string, 0, len(lines)+len(sectionLines)+2)
	for index, group := range groups {
		if index > 0 {
			out = append(out, "")
		}
		out = append(out, group...)
	}
	result := strings.Join(out, "\n")
	if hadTrailingNewline || found {
		result += "\n"
	}
	return result
}

// locateOperationsBlock finds the line range [start, end) of the existing
// Operations block: a generated marker block first, else the section under
// a plain `## Operations` heading.
func locateOperationsBlock(lines []string) (start, end int, found bool) {
	beginLine := -1
	for index, line := range lines {
		if strings.Contains(line, operationsBeginMark) {
			beginLine = index
			break
		}
	}
	if beginLine >= 0 {
		endLine := beginLine
		for index := beginLine + 1; index < len(lines); index++ {
			if strings.Contains(lines[index], operationsEndMark) {
				endLine = index
				break
			}
		}
		start = beginLine
		// Include the heading directly above the marker (blank lines in
		// between are tolerated).
		for cursor := beginLine - 1; cursor >= 0; cursor-- {
			trimmed := strings.TrimSpace(lines[cursor])
			if trimmed == "" {
				continue
			}
			if trimmed == OperationsHeading {
				start = cursor
			}
			break
		}
		return start, endLine + 1, true
	}

	for index, line := range lines {
		if strings.TrimSpace(line) != OperationsHeading {
			continue
		}
		end = len(lines)
		for next := index + 1; next < len(lines); next++ {
			if topLevelHeadingPattern.MatchString(lines[next]) {
				end = next
				break
			}
		}
		return index, end, true
	}
	return 0, 0, false
}

func trimTrailingBlank(lines []string) []string {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[:end]
}

func trimLeadingBlank(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	return lines[start:]
}
