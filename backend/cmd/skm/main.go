package main

import (
	backendapp "backend-go/app"
	"backend-go/internal/config"
	"backend-go/internal/models"
	dbpkg "backend-go/internal/platform/db"
	"backend-go/internal/service"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"gorm.io/gorm"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "help", "--help", "-h":
		return runHelp(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, backendapp.Version)
		return 0
	case "dashboard":
		return runDashboard(args[1:], stdout, stderr)
	case "providers":
		return runProviders(args[1:], stdout, stderr)
	case "skills":
		return runSkills(args[1:], stdout, stderr)
	case "issues":
		return runIssues(args[1:], stdout, stderr)
	case "scan":
		return runScan(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func runHelp(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "providers":
		printProvidersUsage(stdout)
	case "skills":
		printSkillsUsage(stdout)
	case "scan":
		printScanUsage(stdout)
	default:
		fmt.Fprintf(stderr, "unknown help topic: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
	return 0
}

func runDashboard(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("dashboard", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	summary, err := deps.catalog.Dashboard(context.Background())
	if err != nil {
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, summary, stderr)
	}

	fmt.Fprintf(stdout, "Database: %s\n", deps.cfg.DBDSN)
	fmt.Fprintf(stdout, "Providers: %d total, %d enabled\n", summary.ProviderCount, summary.EnabledProviderCount)
	fmt.Fprintf(stdout, "Skills: %d\n", summary.SkillCount)
	fmt.Fprintf(stdout, "Conflicts: %d\n", summary.ConflictCount)
	fmt.Fprintf(stdout, "Issues: %d\n", summary.IssueCount)
	fmt.Fprintf(stdout, "Scans in last 24h: %d\n", summary.RecentScanCount)
	return 0
}

func runProviders(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && isHelpToken(args[0]) {
		printProvidersUsage(stdout)
		return 0
	}

	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runProvidersList(args, stdout, stderr)
	}

	switch args[0] {
	case "help":
		printProvidersUsage(stdout)
		return 0
	case "add":
		return runProvidersAdd(args[1:], stdout, stderr)
	case "update":
		return runProvidersUpdate(args[1:], stdout, stderr)
	case "delete":
		return runProvidersDelete(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown providers subcommand: %s\n", args[0])
		return 2
	}
}

func runProvidersList(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("providers", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	providers, err := deps.catalog.ListProviders(context.Background())
	if err != nil {
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, providers, stderr)
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ZID\tENABLED\tPRIORITY\tNAME\tTYPE\tSCAN\tSTATUS\tROOT")
	for _, provider := range providers {
		fmt.Fprintf(tw, "%s\t%t\t%d\t%s\t%s\t%s\t%s\t%s\n",
			provider.Zid,
			provider.Enabled,
			provider.Priority,
			provider.Name,
			provider.Type,
			provider.ScanMode,
			provider.LastScanStatus,
			provider.RootPath,
		)
	}
	_ = tw.Flush()
	return 0
}

func runProvidersAdd(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("providers add", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	name := fs.String("name", "", "provider name")
	providerType := fs.String("type", "", "provider type")
	rootPath := fs.String("root", "", "provider root path")
	icon := fs.String("icon", "", "provider icon")
	description := fs.String("description", "", "provider description")
	scanMode := fs.String("scan-mode", "recursive", "scan mode: recursive or shallow")
	enabled := fs.Bool("enabled", true, "whether provider is enabled")
	priority := fs.Int("priority", 100, "provider priority")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	provider, err := deps.catalog.CreateProvider(context.Background(), service.ProviderInput{
		Name:        *name,
		Type:        *providerType,
		RootPath:    *rootPath,
		Icon:        *icon,
		Description: *description,
		ScanMode:    *scanMode,
		Enabled:     *enabled,
		Priority:    *priority,
	})
	if err != nil {
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, provider, stderr)
	}

	printProviderDetails(stdout, provider)
	return 0
}

func runProvidersUpdate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("providers update", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	name := fs.String("name", "", "provider name")
	providerType := fs.String("type", "", "provider type")
	rootPath := fs.String("root", "", "provider root path")
	icon := fs.String("icon", "", "provider icon")
	description := fs.String("description", "", "provider description")
	scanMode := fs.String("scan-mode", "", "scan mode: recursive or shallow")
	enabled := fs.String("enabled", "", "provider enabled state: true or false")
	priority := fs.String("priority", "", "provider priority")
	providerZid, err := parseSinglePositional(fs, args)
	if err != nil {
		return 2
	}
	if providerZid == "" {
		fmt.Fprintln(stderr, "usage: skm providers update <provider-zid> [flags]")
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	existing, err := deps.catalog.GetProvider(context.Background(), providerZid)
	if err != nil {
		if errors.Is(err, service.ErrProviderNotFound) {
			fmt.Fprintf(stderr, "provider not found: %s\n", providerZid)
			return 1
		}
		return printError(stderr, err)
	}

	input := service.ProviderInput{
		Name:        existing.Name,
		Type:        existing.Type,
		RootPath:    existing.RootPath,
		Icon:        existing.Icon,
		Description: existing.Description,
		ScanMode:    existing.ScanMode,
		Enabled:     existing.Enabled,
		Priority:    existing.Priority,
	}
	if *name != "" {
		input.Name = *name
	}
	if *providerType != "" {
		input.Type = *providerType
	}
	if *rootPath != "" {
		input.RootPath = *rootPath
	}
	if *icon != "" {
		input.Icon = *icon
	}
	if *description != "" {
		input.Description = *description
	}
	if *scanMode != "" {
		input.ScanMode = *scanMode
	}
	if *enabled != "" {
		parsed, err := strconv.ParseBool(*enabled)
		if err != nil {
			fmt.Fprintf(stderr, "invalid --enabled value: %s\n", *enabled)
			return 2
		}
		input.Enabled = parsed
	}
	if *priority != "" {
		parsed, err := strconv.Atoi(*priority)
		if err != nil {
			fmt.Fprintf(stderr, "invalid --priority value: %s\n", *priority)
			return 2
		}
		input.Priority = parsed
	}

	provider, err := deps.catalog.UpdateProvider(context.Background(), providerZid, input)
	if err != nil {
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, provider, stderr)
	}

	printProviderDetails(stdout, provider)
	return 0
}

func runProvidersDelete(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("providers delete", stderr)
	providerZid, err := parseSinglePositional(fs, args)
	if err != nil {
		return 2
	}
	if providerZid == "" {
		fmt.Fprintln(stderr, "usage: skm providers delete <provider-zid>")
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	if err := deps.catalog.DeleteProvider(context.Background(), providerZid); err != nil {
		if errors.Is(err, service.ErrProviderNotFound) {
			fmt.Fprintf(stderr, "provider not found: %s\n", providerZid)
			return 1
		}
		return printError(stderr, err)
	}

	fmt.Fprintf(stdout, "Deleted provider: %s\n", providerZid)
	return 0
}

func runSkills(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && isHelpToken(args[0]) {
		printSkillsUsage(stdout)
		return 0
	}

	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runSkillsList(args, stdout, stderr)
	}

	switch args[0] {
	case "help":
		printSkillsUsage(stdout)
		return 0
	case "get":
		return runSkillsGet(args[1:], stdout, stderr)
	case "to":
		return runSkillsTo(args[1:], stdout, stderr)
	case "delete":
		return runSkillsDelete(args[1:], stdout, stderr)
	case "link":
		return runSkillsAttach(args[1:], stdout, stderr, "attach")
	case "move":
		return runSkillsAttach(args[1:], stdout, stderr, "move")
	case "sync":
		return runSkillsSync(args[1:], stdout, stderr)
	case "sync-copies":
		return runSkillsSyncCopies(args[1:], stdout, stderr)
	case "exec":
		return runSkillsExec(args[1:], stdout, stderr)
	case "execs":
		return runSkillsExecs(args[1:], stdout, stderr)
	case "gen-operations":
		return runSkillsGenOperations(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown skills subcommand: %s\n", args[0])
		return 2
	}
}

func runSkillsList(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("skills", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	query := fs.String("query", "", "fuzzy search skills by name, slug, tags, category, provider, or summary")
	queryShort := fs.String("q", "", "alias for --query")
	provider := fs.String("provider", "", "provider zid or name")
	category := fs.String("category", "", "filter by category")
	tag := fs.String("tag", "", "filter by tag")
	status := fs.String("status", "", "filter by status")
	sortBy := fs.String("sort", "name", "sort by name, provider, status, lastScanned")
	conflict := fs.String("conflict", "", "filter by conflict: true or false")
	digest := fs.Bool("digest", false, "compact one-line-per-skill digest for LLM context injection")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	var conflictValue *bool
	if *conflict != "" {
		parsed, err := strconv.ParseBool(*conflict)
		if err != nil {
			fmt.Fprintf(stderr, "invalid --conflict value: %s\n", *conflict)
			return 2
		}
		conflictValue = &parsed
	}

	queryText := strings.TrimSpace(*query)
	if shortText := strings.TrimSpace(*queryShort); shortText != "" {
		if queryText != "" && queryText != shortText {
			fmt.Fprintln(stderr, "cannot specify both --query and -q with different values")
			return 2
		}
		queryText = shortText
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	skills, err := deps.catalog.ListSkills(context.Background(), service.SkillListFilters{
		Query:    queryText,
		Provider: *provider,
		Category: *category,
		Tag:      *tag,
		Status:   *status,
		Sort:     *sortBy,
		Conflict: conflictValue,
	})
	if err != nil {
		return printError(stderr, err)
	}

	if *digest {
		renderSkillsDigest(stdout, skills, deps.cfg.DBDSN)
		return 0
	}

	if *jsonOutput {
		return writeJSON(stdout, skills, stderr)
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ZID\tPROVIDER\tNAME\tSTATUS\tCONFLICT\tROOT")
	for _, skill := range skills {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%t\t%s\n",
			skill.Zid,
			skill.Provider.Name,
			skill.Name,
			skill.Status,
			skill.IsConflict,
			skill.RootPath,
		)
	}
	_ = tw.Flush()
	fmt.Fprintf(stdout, "\nTotal skills: %d\n", len(skills))
	return 0
}

func runSkillsGet(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && isHelpToken(args[0]) {
		printSkillsGetUsage(stdout)
		return 0
	}

	fs := newFlagSet("skills get", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	files := fs.Bool("files", false, "list all skill files with ls-style details")
	filesShort := fs.Bool("f", false, "alias for --files")
	commands := fs.Bool("commands", false, "list executable commands declared in package.json")
	positionals, err := parsePositionals(fs, args, 2)
	if err != nil || len(positionals) == 0 {
		fmt.Fprintln(stderr, "usage: skm skills get <skill-zid> [path] [--files] [--commands] [--json]")
		return 2
	}
	skillZid := positionals[0]
	filePath := ""
	if len(positionals) > 1 {
		filePath = strings.TrimSpace(positionals[1])
	}
	showFiles := *files || *filesShort
	if filePath != "" && showFiles {
		fmt.Fprintln(stderr, "--files cannot be combined with a file path")
		return 2
	}
	if *commands && (showFiles || filePath != "") {
		fmt.Fprintln(stderr, "--commands cannot be combined with --files or a file path")
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	ctx := context.Background()
	skill, err := deps.catalog.GetSkill(ctx, skillZid)
	if err != nil {
		if errors.Is(err, service.ErrSkillNotFound) {
			fmt.Fprintf(stderr, "skill not found: %s\n", skillZid)
			return 1
		}
		return printError(stderr, err)
	}

	if filePath != "" {
		return runSkillsGetFile(ctx, stdout, stderr, deps.catalog, skill, filePath, *jsonOutput)
	}

	if *commands {
		return runSkillsGetCommands(ctx, stdout, deps.exec, skill, *jsonOutput)
	}

	if *jsonOutput {
		if showFiles {
			nodes, filesErr := deps.catalog.GetSkillFiles(ctx, skillZid)
			if filesErr != nil {
				return printError(stderr, filesErr)
			}
			return writeJSON(stdout, fileListEntries(nodes), stderr)
		}
		return writeJSON(stdout, skill, stderr)
	}

	printSkillDetails(stdout, skill)
	fmt.Fprintln(stdout)
	printSeparator(stdout)

	var nextSteps []nextStep
	if showFiles {
		nodes, filesErr := deps.catalog.GetSkillFiles(ctx, skillZid)
		if filesErr != nil {
			return printError(stderr, filesErr)
		}
		printFileListing(stdout, nodes)
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, fileListingSummary(nodes))
		nextSteps = suggestFromFileListing(skill, nodes)
	} else {
		content, note := readSkillMarkdown(skill)
		fmt.Fprintln(stdout, content)
		if note != "" {
			fmt.Fprintf(stdout, "\n%s\n", note)
		}
		nodes, filesErr := deps.catalog.GetSkillFiles(ctx, skillZid)
		if filesErr != nil {
			nodes = nil
		}
		nextSteps = suggestFromOverview(skill, nodes)
	}
	fmt.Fprintln(stdout)
	printSeparator(stdout)
	fmt.Fprintf(stdout, "Path: %s\n", skill.RootPath)
	printNextSteps(stdout, nextSteps)
	return 0
}

func runSkillsGetFile(ctx context.Context, stdout, stderr io.Writer, catalog *service.CatalogService, skill *models.Skill, relativePath string, jsonOutput bool) int {
	cleanPath := filepath.ToSlash(filepath.Clean(relativePath))
	if filepath.IsAbs(cleanPath) {
		if rel, err := filepath.Rel(skill.RootPath, filepath.FromSlash(cleanPath)); err == nil && !strings.HasPrefix(rel, "..") {
			cleanPath = filepath.ToSlash(rel)
		}
	}

	detail, err := catalog.GetSkillFileDetail(ctx, skill.Zid, cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "file not found in skill %s: %s\n", skill.Zid, cleanPath)
			return 1
		}
		return printError(stderr, err)
	}

	if jsonOutput {
		view := struct {
			SkillZid  string `json:"skillZid"`
			SkillName string `json:"skillName"`
			*service.SkillFileDetail
		}{skill.Zid, skill.Name, detail}
		return writeJSON(stdout, view, stderr)
	}

	fileType := "text"
	switch {
	case detail.IsDir:
		fileType = "directory"
	case detail.Binary:
		fileType = "binary"
	}
	printFileInfo(stdout, skill, detail, fileType)

	if detail.Binary {
		printNextSteps(stdout, suggestFromFileView(skill, nil))
		return 0
	}

	fmt.Fprintln(stdout)
	printSeparator(stdout)
	if detail.IsDir {
		printDirectoryChildren(stdout, detail.Children)
	} else {
		fmt.Fprintln(stdout, strings.TrimRight(detail.Content, "\n"))
	}
	printNextSteps(stdout, suggestFromFileView(skill, detail))
	return 0
}

// runSkillsGetCommands lists the executable commands declared in the skill's
// package.json via the shared exec-service view (live parse from disk,
// catalog snapshot as fallback).
func runSkillsGetCommands(ctx context.Context, stdout io.Writer, exec *service.ExecService, skill *models.Skill, jsonOutput bool) int {
	view, err := exec.Commands(ctx, skill.Zid)
	if err != nil {
		return printError(os.Stderr, err)
	}

	if jsonOutput {
		return writeJSON(stdout, view, os.Stderr)
	}

	fmt.Fprintf(stdout, "Skill: %s (%s)\n", skill.Name, skill.Zid)
	fmt.Fprintf(stdout, "Manifest: %s\n", service.ManifestPath(view.SkillRoot))
	if view.Note != "" {
		fmt.Fprintf(stdout, "Note: %s\n", view.Note)
	}
	if len(view.Commands) == 0 {
		fmt.Fprintln(stdout, "Commands: none declared")
		return 0
	}

	if len(view.RuntimeEnv) > 0 {
		fmt.Fprintf(stdout, "Required env: %s\n", strings.Join(view.RuntimeEnv, ", "))
	}
	if view.Setup != "" {
		fmt.Fprintf(stdout, "Setup command: %s\n", view.Setup)
	}
	fmt.Fprintln(stdout)

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COMMAND\tDESCRIPTION\tCONFIRM\tTIMEOUT\tENV\tINPUT")
	for _, command := range view.Commands {
		description := oneLine(command.Description)
		if description == "" {
			description = "-"
		}
		confirm := "-"
		if command.Confirm != "" {
			confirm = "yes"
		}
		timeout := "-"
		if command.TimeoutSeconds > 0 {
			timeout = fmt.Sprintf("%ds", command.TimeoutSeconds)
		}
		env := "-"
		if len(command.Env) > 0 {
			env = strings.Join(command.Env, ",")
		}
		input := "-"
		if command.InputVia != "" {
			input = command.InputVia
			if command.HasInputSchema {
				input += "+schema"
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", command.Name, description, confirm, timeout, env, input)
	}
	_ = tw.Flush()

	fmt.Fprintln(stdout)
	printSeparator(stdout)
	fmt.Fprintf(stdout, "Path: %s\n", skill.RootPath)
	steps := make([]nextStep, 0, len(view.Commands))
	for _, command := range view.Commands {
		steps = append(steps, nextStep{
			command: fmt.Sprintf("skm skills exec %s %s --dry-run", skill.Zid, command.Name),
			comment: "preview running " + command.Name,
		})
		if len(steps) >= 3 {
			break
		}
	}
	printNextSteps(stdout, steps)
	return 0
}

func printFileInfo(stdout io.Writer, skill *models.Skill, detail *service.SkillFileDetail, fileType string) {
	fmt.Fprintf(stdout, "File: %s\n", detail.Path)
	fmt.Fprintf(stdout, "Skill: %s (%s)\n", skill.Name, skill.Zid)
	fmt.Fprintf(stdout, "Type: %s\n", fileType)
	if detail.IsDir {
		fmt.Fprintf(stdout, "Children: %d\n", len(detail.Children))
	} else {
		fmt.Fprintf(stdout, "Size: %s (%d bytes)\n", humanSize(detail.Size), detail.Size)
	}
	if detail.ModifiedAt != nil {
		fmt.Fprintf(stdout, "Modified: %s\n", formatTime(*detail.ModifiedAt))
	}
	fmt.Fprintf(stdout, "Absolute: %s\n", filepath.Join(skill.RootPath, filepath.FromSlash(detail.Path)))
	if detail.Binary {
		fmt.Fprintln(stdout, "Content: not shown (binary or non-text file)")
	}
}

func printSkillsGetUsage(out io.Writer) {
	fmt.Fprintln(out, "# skm skills get")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Show skill details with the SKILL.md content, a directory file tree, or one file's content.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  skm skills get <skill-zid> [--json]            skill info + SKILL.md content + directory path")
	fmt.Fprintln(out, "  skm skills get <skill-zid> --files             skill info + ls-style listing of all files")
	fmt.Fprintln(out, "  skm skills get <skill-zid> --commands          executable commands declared in package.json")
	fmt.Fprintln(out, "  skm skills get <skill-zid> <path> [--json]     file info + file content")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Binary or non-text files are not printed; the file info states that the content is not shown.")
	fmt.Fprintln(out, "Each text view ends with recommended Next commands.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  -f, --files   list all skill files (ls -l style, paths relative to the skill root) instead of the SKILL.md content")
	fmt.Fprintln(out, "  --commands    list executable commands declared in the skill's package.json (scripts + skm annotations)")
	fmt.Fprintln(out, "  --json        output JSON")
	fmt.Fprintln(out, "  -h, --help    help for skm skills get")
}

func readSkillMarkdown(skill *models.Skill) (string, string) {
	if skill.SkillMdPath != "" {
		if data, err := os.ReadFile(skill.SkillMdPath); err == nil {
			return strings.TrimRight(string(data), "\n"), ""
		}
	}
	if strings.TrimSpace(skill.RawMarkdown) != "" {
		return strings.TrimRight(skill.RawMarkdown, "\n"), "(SKILL.md not found on disk; showing catalog snapshot)"
	}
	return "(no SKILL.md content available)", ""
}

func printSeparator(stdout io.Writer) {
	fmt.Fprintln(stdout, strings.Repeat("─", 60))
}

type fileListEntry struct {
	Path       string     `json:"path"`
	IsDir      bool       `json:"isDir"`
	Mode       string     `json:"mode"`
	Size       int64      `json:"size,omitempty"`
	ModifiedAt *time.Time `json:"modifiedAt,omitempty"`
}

func flattenFileNodes(nodes []service.FileNode) []service.FileNode {
	var flat []service.FileNode
	for _, node := range nodes {
		flat = append(flat, node)
		if node.IsDir {
			flat = append(flat, flattenFileNodes(node.Children)...)
		}
	}
	return flat
}

func fileListEntries(nodes []service.FileNode) []fileListEntry {
	flat := flattenFileNodes(nodes)
	entries := make([]fileListEntry, 0, len(flat))
	for _, node := range flat {
		entry := fileListEntry{
			Path:       node.Path,
			IsDir:      node.IsDir,
			Mode:       nodeModeString(node),
			ModifiedAt: node.ModifiedAt,
		}
		if !node.IsDir {
			entry.Size = node.Size
		}
		entries = append(entries, entry)
	}
	return entries
}

func nodeModeString(node service.FileNode) string {
	if node.Mode != 0 {
		return node.Mode.String()
	}
	if node.IsDir {
		return "drwxr-xr-x"
	}
	return "-rw-r--r--"
}

// printFileListing renders an ls -l style listing with paths relative to the
// skill root, one line per file or directory.
func printFileListing(stdout io.Writer, nodes []service.FileNode) {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, node := range flattenFileNodes(nodes) {
		name := node.Path
		size := "-"
		if node.IsDir {
			name += "/"
		} else {
			size = humanSize(node.Size)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", nodeModeString(node), size, formatLsTime(node.ModifiedAt), name)
	}
	_ = tw.Flush()
}

func formatLsTime(value *time.Time) string {
	if value == nil {
		return "-"
	}
	local := value.Local()
	if local.Year() == time.Now().Year() {
		return local.Format("Jan _2 15:04")
	}
	return local.Format("Jan _2 2006")
}

func fileListingSummary(nodes []service.FileNode) string {
	files, directories := 0, 0
	var totalSize int64
	for _, node := range flattenFileNodes(nodes) {
		if node.IsDir {
			directories++
			continue
		}
		files++
		totalSize += node.Size
	}
	return fmt.Sprintf("Total: %d files, %d directories, %s", files, directories, humanSize(totalSize))
}

type nextStep struct {
	command string
	comment string
}

func printNextSteps(stdout io.Writer, steps []nextStep) {
	if len(steps) == 0 {
		return
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Next:")
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, step := range steps {
		fmt.Fprintf(tw, "  %s\t# %s\n", step.command, step.comment)
	}
	_ = tw.Flush()
}

func suggestFromOverview(skill *models.Skill, nodes []service.FileNode) []nextStep {
	steps := []nextStep{{
		command: fmt.Sprintf("skm skills get %s --files", skill.Zid),
		comment: "list all files in the skill",
	}}
	if example := pickExampleFile(nodes, skillMarkdownExclude(skill)); example != nil {
		steps = append(steps, nextStep{
			command: fmt.Sprintf("skm skills get %s %s", skill.Zid, example.Path),
			comment: "view file content",
		})
	}
	if len(skill.Commands) > 0 {
		steps = append(steps, nextStep{
			command: fmt.Sprintf("skm skills get %s --commands", skill.Zid),
			comment: "list declared executable commands",
		})
	}
	return steps
}

func suggestFromFileListing(skill *models.Skill, nodes []service.FileNode) []nextStep {
	examples := pickExampleFiles(nodes, skillMarkdownExclude(skill), 2)
	if len(examples) == 0 {
		return []nextStep{{
			command: fmt.Sprintf("skm skills get %s <path>", skill.Zid),
			comment: "view file content",
		}}
	}
	steps := make([]nextStep, 0, len(examples))
	for _, node := range examples {
		steps = append(steps, nextStep{
			command: fmt.Sprintf("skm skills get %s %s", skill.Zid, node.Path),
			comment: "view file content",
		})
	}
	return steps
}

func suggestFromFileView(skill *models.Skill, detail *service.SkillFileDetail) []nextStep {
	var steps []nextStep
	if detail != nil && detail.IsDir {
		for _, node := range pickExampleFiles(detail.Children, nil, 1) {
			steps = append(steps, nextStep{
				command: fmt.Sprintf("skm skills get %s %s", skill.Zid, node.Path),
				comment: "view file content",
			})
		}
	}
	steps = append(steps,
		nextStep{
			command: fmt.Sprintf("skm skills get %s", skill.Zid),
			comment: "skill overview and SKILL.md content",
		},
		nextStep{
			command: fmt.Sprintf("skm skills get %s --files", skill.Zid),
			comment: "list all files in the skill",
		},
	)
	return steps
}

func skillMarkdownExclude(skill *models.Skill) map[string]bool {
	exclude := map[string]bool{}
	if skill.SkillMdPath != "" {
		if rel, err := filepath.Rel(skill.RootPath, skill.SkillMdPath); err == nil {
			exclude[filepath.ToSlash(rel)] = true
		}
	}
	return exclude
}

var preferredExampleExtensions = map[string]int{
	".md": 1, ".markdown": 1,
	".json": 2, ".yaml": 2, ".yml": 2, ".toml": 2,
	".txt": 3,
	".sh":  4, ".py": 4, ".js": 4, ".ts": 4,
}

const maxExampleFileSize = 64 * 1024

// pickExampleFiles chooses up to limit small, human-readable files suitable
// for recommending as the next thing to view.
func pickExampleFiles(nodes []service.FileNode, exclude map[string]bool, limit int) []service.FileNode {
	type candidate struct {
		node service.FileNode
		rank int
	}
	var candidates []candidate
	for _, node := range flattenFileNodes(nodes) {
		if node.IsDir || exclude[node.Path] || node.Size > maxExampleFileSize {
			continue
		}
		rank, known := preferredExampleExtensions[strings.ToLower(filepath.Ext(node.Path))]
		if !known {
			rank = 5
		}
		candidates = append(candidates, candidate{node: node, rank: rank})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].rank != candidates[j].rank {
			return candidates[i].rank < candidates[j].rank
		}
		return candidates[i].node.Path < candidates[j].node.Path
	})
	picked := make([]service.FileNode, 0, limit)
	for _, item := range candidates {
		if len(picked) == limit {
			break
		}
		picked = append(picked, item.node)
	}
	return picked
}

func pickExampleFile(nodes []service.FileNode, exclude map[string]bool) *service.FileNode {
	picked := pickExampleFiles(nodes, exclude, 1)
	if len(picked) == 0 {
		return nil
	}
	return &picked[0]
}

func printDirectoryChildren(stdout io.Writer, nodes []service.FileNode) {
	for _, node := range nodes {
		if node.IsDir {
			fmt.Fprintf(stdout, "%s/\n", node.Name)
			continue
		}
		fmt.Fprintf(stdout, "%s  %s\n", node.Name, humanSize(node.Size))
	}
}

func humanSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func runSkillsTo(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("skills to", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	providerPath := fs.String("provider-path", "", "provider root to reuse or create; must be the current directory or a parent directory")
	var directories multiStringFlag
	var includePatterns multiStringFlag
	var excludePatterns multiStringFlag
	fs.Var(&directories, "directory", "target directory to append into .to; repeatable")
	fs.Var(&includePatterns, "include", "include glob pattern to set on .to; repeatable")
	fs.Var(&excludePatterns, "exclude", "exclude glob pattern to set on .to; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	rootPath, err := os.Getwd()
	if err != nil {
		return printError(stderr, err)
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	result, err := deps.catalog.ConfigureSkillTo(context.Background(), service.SkillToInput{
		RootPath:     rootPath,
		ProviderPath: strings.TrimSpace(*providerPath),
		Directories:  directories.Values(),
		Include:      includePatterns.Values(),
		Exclude:      excludePatterns.Values(),
	})
	if err != nil {
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, result, stderr)
	}

	fmt.Fprintf(stdout, ".to updated: %s\n", result.FilePath)
	fmt.Fprintf(stdout, "Root: %s\n", result.RootPath)
	if result.Provider != nil {
		fmt.Fprintf(stdout, "Provider: %s (%s)\n", result.Provider.Name, result.Provider.Zid)
		fmt.Fprintf(stdout, "Provider root: %s\n", result.Provider.RootPath)
		if result.ProviderCreated {
			fmt.Fprintln(stdout, "Provider status: created")
		} else {
			fmt.Fprintln(stdout, "Provider status: existing")
		}
	}
	if result.Relation != nil {
		fmt.Fprintf(stdout, "Directories: %s\n", strings.Join(result.Relation.Directories, ", "))
		fmt.Fprintf(stdout, "Include: %s\n", strings.Join(result.Relation.Include, ", "))
		if len(result.Relation.Exclude) > 0 {
			fmt.Fprintf(stdout, "Exclude: %s\n", strings.Join(result.Relation.Exclude, ", "))
		}
	}
	return 0
}

func runSkillsDelete(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("skills delete", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	force := fs.Bool("force", false, "force delete source skill even if attached copies exist")
	skillZid, err := parseSinglePositional(fs, args)
	if err != nil {
		return 2
	}
	if skillZid == "" {
		fmt.Fprintln(stderr, "usage: skm skills delete <skill-zid> [--force]")
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	result, err := deps.catalog.DeleteSkill(context.Background(), skillZid, service.SkillDeleteInput{Force: *force})
	if err != nil {
		if errors.Is(err, service.ErrSkillNotFound) {
			fmt.Fprintf(stderr, "skill not found: %s\n", skillZid)
			return 1
		}
		return printError(stderr, err)
	}
	job, err := deps.scan.ScanProviderByZid(context.Background(), result.Provider.Zid)
	if err != nil {
		return printError(stderr, err)
	}
	result.Job = job

	if *jsonOutput {
		return writeJSON(stdout, result, stderr)
	}

	fmt.Fprintf(stdout, "Deleted skill: %s\n", result.SkillZid)
	fmt.Fprintf(stdout, "Mode: %s\n", result.DeleteMode)
	fmt.Fprintf(stdout, "Path: %s\n", result.DeletedPath)
	if result.CopyCount > 0 {
		fmt.Fprintf(stdout, "Attached copies: %d\n", result.CopyCount)
	}
	fmt.Fprintf(stdout, "Rescan job: %s\n", job.Zid)
	return 0
}

func runSkillsAttach(args []string, stdout, stderr io.Writer, mode string) int {
	commandName := "skills link"
	if mode == "move" {
		commandName = "skills move"
	}
	fs := newFlagSet(commandName, stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	targetProvider := fs.String("to", "", "target provider zid")
	skillZid, err := parseSinglePositional(fs, args)
	if err != nil {
		return 2
	}
	if skillZid == "" || strings.TrimSpace(*targetProvider) == "" {
		fmt.Fprintf(stderr, "usage: skm %s <skill-zid> --to <provider-zid>\n", commandName)
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	result, err := deps.catalog.AttachSkill(context.Background(), skillZid, service.SkillAttachInput{
		TargetProviderZid: *targetProvider,
		Mode:              mode,
	})
	if err != nil {
		if errors.Is(err, service.ErrSkillNotFound) {
			fmt.Fprintf(stderr, "skill not found: %s\n", skillZid)
			return 1
		}
		if errors.Is(err, service.ErrProviderNotFound) {
			fmt.Fprintf(stderr, "target provider not found: %s\n", *targetProvider)
			return 1
		}
		return printError(stderr, err)
	}

	jobs := make([]service.SkillAttachScanJob, 0, 2)
	if result.Mode == "move" {
		sourceJob, err := deps.scan.ScanProviderByZid(context.Background(), result.SourceProvider.Zid)
		if err != nil {
			return printError(stderr, err)
		}
		jobs = append(jobs, service.SkillAttachScanJob{ProviderZid: result.SourceProvider.Zid, Job: *sourceJob})
	}
	targetJob, err := deps.scan.ScanProviderByZid(context.Background(), result.TargetProvider.Zid)
	if err != nil {
		return printError(stderr, err)
	}
	jobs = append(jobs, service.SkillAttachScanJob{ProviderZid: result.TargetProvider.Zid, Job: *targetJob})
	result.Jobs = jobs

	if *jsonOutput {
		return writeJSON(stdout, result, stderr)
	}

	label := "Link"
	if mode == "move" {
		label = "Move"
	}
	fmt.Fprintf(stdout, "%s skill: %s\n", label, result.SkillZid)
	fmt.Fprintf(stdout, "Source: %s\n", result.SourcePath)
	fmt.Fprintf(stdout, "Target: %s\n", result.TargetPath)
	fmt.Fprintf(stdout, "Target provider: %s (%s)\n", result.TargetProvider.Name, result.TargetProvider.Zid)
	for _, job := range result.Jobs {
		fmt.Fprintf(stdout, "Rescan %s: %s\n", job.ProviderZid, job.Job.Zid)
	}
	return 0
}

func runSkillsSync(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("skills sync", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	skillZid, err := parseSinglePositional(fs, args)
	if err != nil {
		return 2
	}
	if skillZid == "" {
		fmt.Fprintln(stderr, "usage: skm skills sync <skill-zid>")
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	result, err := deps.catalog.SyncSkill(context.Background(), skillZid)
	if err != nil {
		if errors.Is(err, service.ErrSkillNotFound) {
			fmt.Fprintf(stderr, "skill not found: %s\n", skillZid)
			return 1
		}
		return printError(stderr, err)
	}
	job, err := deps.scan.ScanProviderByZid(context.Background(), result.Provider.Zid)
	if err != nil {
		return printError(stderr, err)
	}
	result.Job = job

	if *jsonOutput {
		return writeJSON(stdout, result, stderr)
	}

	fmt.Fprintf(stdout, "Synced skill: %s\n", result.SkillZid)
	fmt.Fprintf(stdout, "Source: %s\n", result.SourcePath)
	fmt.Fprintf(stdout, "Target: %s\n", result.TargetPath)
	fmt.Fprintf(stdout, "Rescan job: %s\n", job.Zid)
	return 0
}

func runSkillsSyncCopies(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("skills sync-copies", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	skillZid, err := parseSinglePositional(fs, args)
	if err != nil {
		return 2
	}
	if skillZid == "" {
		fmt.Fprintln(stderr, "usage: skm skills sync-copies <skill-zid>")
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	result, err := deps.catalog.SyncSkillCopies(context.Background(), skillZid)
	if err != nil {
		if errors.Is(err, service.ErrSkillNotFound) {
			fmt.Fprintf(stderr, "skill not found: %s\n", skillZid)
			return 1
		}
		return printError(stderr, err)
	}

	providerZids := result.ScannedProviderZids
	if len(providerZids) == 0 {
		providerZids = []string{result.Provider.Zid}
	}
	for _, providerZid := range providerZids {
		job, scanErr := deps.scan.ScanProviderByZid(context.Background(), providerZid)
		if scanErr != nil {
			return printError(stderr, scanErr)
		}
		if providerZid == result.Provider.Zid {
			result.Job = job
		}
	}

	if *jsonOutput {
		return writeJSON(stdout, result, stderr)
	}

	fmt.Fprintf(stdout, "Synced source skill: %s\n", result.SkillZid)
	fmt.Fprintf(stdout, "Source: %s\n", result.SourcePath)
	fmt.Fprintf(stdout, "Copies: %d\n", len(result.Copies))
	for _, copy := range result.Copies {
		if copy.SkillZid != "" {
			fmt.Fprintf(stdout, "  - %s (%s)\n", copy.TargetPath, copy.SkillZid)
		} else {
			fmt.Fprintf(stdout, "  - %s\n", copy.TargetPath)
		}
	}
	if result.Job != nil {
		fmt.Fprintf(stdout, "Rescan job: %s\n", result.Job.Zid)
	}
	return 0
}

// splitPassthrough separates the arguments after the first bare "--", which
// are passed through to the executed command untouched.
func splitPassthrough(args []string) (head, passthrough []string) {
	for index, arg := range args {
		if arg == "--" {
			return args[:index], args[index+1:]
		}
	}
	return args, nil
}

func readInputValue(spec string, stdin io.Reader) ([]byte, error) {
	switch {
	case spec == "-":
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("read input from stdin: %w", err)
		}
		return data, nil
	case strings.HasPrefix(spec, "@"):
		data, err := os.ReadFile(strings.TrimPrefix(spec, "@"))
		if err != nil {
			return nil, fmt.Errorf("read input file: %w", err)
		}
		return data, nil
	default:
		return []byte(spec), nil
	}
}

func runSkillsExec(args []string, stdout, stderr io.Writer) int {
	head, passthrough := splitPassthrough(args)
	if len(head) > 0 && isHelpToken(head[0]) {
		printSkillsExecUsage(stdout)
		return 0
	}

	fs := newFlagSet("skills exec", stderr)
	inputSpec := fs.String("input", "", "structured JSON input: literal, @file, or - for stdin")
	var envFlag repeatStringFlag
	fs.Var(&envFlag, "env", "environment variable KEY=VAL to inject (repeatable)")
	assumeYes := fs.Bool("yes", false, "required for commands that declare confirm")
	dryRun := fs.Bool("dry-run", false, "print the resolved command without executing")
	jsonOutput := fs.Bool("json", false, "structured result output")
	timeoutSeconds := fs.Int("timeout", 0, "override the manifest timeout, in seconds")
	isolate := fs.Bool("isolate", false, "run in a materialized cache copy, keeping the source directory clean")
	pin := fs.String("pin", "", "run the version whose source hash starts with this value (8-64 hex chars)")
	setupOnly := fs.Bool("setup", false, "run only the manifest's runtime.setup command (idempotent)")
	forceSetup := fs.Bool("force", false, "with --setup: re-run setup even when the completion marker is fresh")

	// Flags may appear before or after the positionals, so parse first and
	// validate the shape once *setupOnly is known.
	positionals, err := parsePositionals(fs, head, 2)
	if err != nil {
		fmt.Fprintln(stderr, "usage: skm skills exec <skill-zid> <command> [flags] [-- args...]")
		return 2
	}
	if *setupOnly {
		if len(positionals) != 1 {
			fmt.Fprintln(stderr, "usage: skm skills exec <skill-zid> --setup [--isolate] [--force] [--pin hash] [--dry-run] [--json]")
			return 2
		}
		if len(passthrough) > 0 {
			fmt.Fprintln(stderr, "--setup takes no command or pass-through arguments")
			return 2
		}
	} else if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: skm skills exec <skill-zid> <command> [flags] [-- args...]")
		return 2
	}

	for _, entry := range envFlag.Values() {
		if !strings.Contains(entry, "=") {
			fmt.Fprintf(stderr, "invalid --env value %q: expected KEY=VAL\n", entry)
			return 2
		}
	}

	var inputJSON []byte
	if strings.TrimSpace(*inputSpec) != "" {
		inputJSON, err = readInputValue(*inputSpec, os.Stdin)
		if err != nil {
			return printError(stderr, err)
		}
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	ctx := context.Background()
	var result *service.ExecResult
	if *setupOnly {
		request := &service.SetupRequest{
			SkillZid: positionals[0],
			Isolate:  *isolate,
			Force:    *forceSetup,
			DryRun:   *dryRun,
			Pin:      *pin,
			Trigger:  "cli",
		}
		if !*jsonOutput {
			request.Stdout = stdout
			request.Stderr = stderr
		}
		result, err = deps.exec.RunSetup(ctx, request)
	} else {
		request := &service.ExecRequest{
			SkillZid:        positionals[0],
			Command:         positionals[1],
			Args:            passthrough,
			InputJSON:       inputJSON,
			Env:             envFlag.Values(),
			AssumeYes:       *assumeYes,
			TimeoutOverride: time.Duration(*timeoutSeconds) * time.Second,
			DryRun:          *dryRun,
			Isolate:         *isolate,
			Pin:             *pin,
			Trigger:         "cli",
		}
		if !*jsonOutput {
			request.Stdout = stdout
			request.Stderr = stderr
		}
		result, err = deps.exec.Exec(ctx, request)
	}
	if err != nil {
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSONAndExit(stdout, stderr, result)
	}

	if result.DryRun {
		printExecPlan(stdout, result)
		return 0
	}
	if result.Aborted == "deps-failed" {
		fmt.Fprintln(stderr, "managed dependency installation failed: the command was not started")
	}
	if result.Setup != nil && result.Setup.Ran {
		if result.Aborted == "setup-failed" {
			fmt.Fprintf(stderr, "setup command %q failed: the command was not started\n", result.Setup.Command)
		}
	}
	if result.TimedOut {
		fmt.Fprintf(stderr, "command timed out after %dms\n", result.DurationMs)
	}
	exitCode := result.ExitCode
	if exitCode < 0 {
		exitCode = 1
	}
	return exitCode
}

// runSkillsExecs lists audited exec invocations (newest first). The HASH
// column shows the source hash a future --pin can replay; REASON explains
// rejected and aborted runs.
func runSkillsExecs(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("skills execs", stderr)
	skillZid := fs.String("skill", "", "only show records of this skill zid")
	limit := fs.Int("limit", 20, "maximum number of records")
	jsonOutput := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	records, err := deps.exec.ListExecRecords(context.Background(), *skillZid, *limit)
	if err != nil {
		return printError(stderr, err)
	}
	if *jsonOutput {
		return writeJSON(stdout, records, stderr)
	}
	if len(records) == 0 {
		fmt.Fprintln(stdout, "(no exec records)")
		return 0
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TIME\tSKILL ZID\tSKILL\tCOMMAND\tTRIGGER\tSTATUS\tEXIT\tMS\tHASH\tREASON")
	for _, record := range records {
		exit := strconv.Itoa(record.ExitCode)
		if record.Status == "rejected" {
			exit = "-"
		}
		hash := record.SourceHash
		if len(hash) > 12 {
			hash = hash[:12]
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			formatTime(record.StartedAt), record.SkillZid, record.SkillName, record.Command,
			record.Trigger, record.Status, exit, record.DurationMs, hash, oneLine(record.Reason))
	}
	_ = tw.Flush()
	return 0
}

func printExecPlan(stdout io.Writer, result *service.ExecResult) {
	plan := result.Plan
	fmt.Fprintf(stdout, "Skill: %s (%s)\n", result.SkillName, result.SkillZid)
	fmt.Fprintf(stdout, "Command: %s\n", result.Command)
	fmt.Fprintf(stdout, "WorkDir: %s (%s)\n", plan.WorkDir, describePlanMode(plan))
	if plan.SourceDir != "" && plan.Mode == "cache" {
		fmt.Fprintf(stdout, "Source: %s\n", plan.SourceDir)
	}
	if plan.Pin != "" {
		fmt.Fprintf(stdout, "Pin: %s\n", plan.Pin)
	}
	fmt.Fprintf(stdout, "Run: %s\n", plan.FormatCommandLine())
	if plan.InputVia != "" {
		fmt.Fprintf(stdout, "Input: %s (%d bytes)\n", plan.InputVia, plan.InputBytes)
	}
	if len(plan.EnvAdditions) > 0 {
		fmt.Fprintf(stdout, "Env: %s\n", strings.Join(plan.EnvAdditions, " "))
	}
	if plan.TimeoutSeconds > 0 {
		fmt.Fprintf(stdout, "Timeout: %ds\n", plan.TimeoutSeconds)
	}
	if plan.Confirm != "" {
		fmt.Fprintf(stdout, "Confirm: %s\n", plan.Confirm)
	}
	if len(plan.Deps) > 0 {
		for _, action := range plan.Deps {
			fmt.Fprintf(stdout, "Deps: %s (will run before setup)\n", action)
		}
	} else if plan.DepsSkipped {
		fmt.Fprintln(stdout, "Deps: managed dependencies up to date, will be skipped")
	}
	if plan.Setup != "" {
		if plan.SetupSkipped {
			fmt.Fprintf(stdout, "Setup: %s (up to date, will be skipped)\n", plan.Setup)
		} else {
			fmt.Fprintf(stdout, "Setup: %s (will run before the command)\n", plan.Setup)
		}
	}
	fmt.Fprintln(stdout, "Dry run: nothing was executed")
}

func describePlanMode(plan *service.ExecPlan) string {
	switch {
	case plan.Mode == "cache" && plan.Materialized:
		return "cache copy, materialized for this run"
	case plan.Mode == "cache" && plan.CacheReused:
		return "cache copy, reused"
	case plan.Mode == "cache":
		return "cache copy, will be materialized"
	default:
		return "source directory"
	}
}

func runSkillsGenOperations(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && isHelpToken(args[0]) {
		printSkillsGenOperationsUsage(stdout)
		return 0
	}

	fs := newFlagSet("skills gen-operations", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	check := fs.Bool("check", false, "exit 1 when SKILL.md would change, without writing")
	all := fs.Bool("all", false, "regenerate for every catalog skill that has a package.json")
	positionals, err := parsePositionals(fs, args, 1)
	if err != nil {
		return 2
	}
	if *all && len(positionals) != 0 {
		fmt.Fprintln(stderr, "--all takes no skill argument")
		return 2
	}
	if !*all && len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: skm skills gen-operations <skill-zid>|--all [--check] [--json]")
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	ctx := context.Background()
	if *all {
		return runSkillsGenOperationsAll(ctx, stdout, stderr, deps.catalog, *check, *jsonOutput)
	}

	skill, err := deps.catalog.GetSkill(ctx, positionals[0])
	if err != nil {
		if errors.Is(err, service.ErrSkillNotFound) {
			fmt.Fprintf(stderr, "skill not found: %s\n", positionals[0])
			return 1
		}
		return printError(stderr, err)
	}

	root := service.SkillOperationsRoot(skill)
	result, err := service.GenerateOperations(root, skill.Zid, skill.Name, *check)
	if err != nil {
		if errors.Is(err, service.ErrManifestNotFound) {
			fmt.Fprintf(stderr, "skill %s declares no executable manifest (missing package.json)\n", skill.Zid)
			return 1
		}
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, result, stderr)
	}

	fmt.Fprintf(stdout, "Skill: %s (%s)\n", result.SkillName, result.SkillZid)
	fmt.Fprintf(stdout, "SKILL.md: %s\n", filepath.Join(result.SkillRoot, "SKILL.md"))
	fmt.Fprintf(stdout, "Commands: %d\n", result.CommandCount)
	switch {
	case *check && result.Changed:
		fmt.Fprintln(stdout, "Check: SKILL.md is out of date with package.json")
		return 1
	case *check:
		fmt.Fprintln(stdout, "Check: SKILL.md is up to date")
	case result.Written:
		fmt.Fprintln(stdout, "Updated the Operations section in SKILL.md")
		fmt.Fprintln(stdout, "Next: run `skm skills sync-copies "+skill.Zid+"` to refresh attached copies")
	default:
		fmt.Fprintln(stdout, "Operations section already up to date")
	}
	return 0
}

func runSkillsGenOperationsAll(ctx context.Context, stdout, stderr io.Writer, catalog *service.CatalogService, check, jsonOutput bool) int {
	skills, err := catalog.ListSkills(ctx, service.SkillListFilters{})
	if err != nil {
		return printError(stderr, err)
	}

	results := make([]service.OperationsResult, 0)
	skipped := 0
	failed := 0
	seenRoots := map[string]struct{}{}
	for index := range skills {
		skill := &skills[index]
		root := service.SkillOperationsRoot(skill)
		if _, seen := seenRoots[root]; seen {
			continue
		}
		seenRoots[root] = struct{}{}
		if _, manifestErr := service.LoadManifest(root); manifestErr != nil {
			skipped++
			continue
		}
		result, genErr := service.GenerateOperations(root, skill.Zid, skill.Name, check)
		if genErr != nil {
			failed++
			fmt.Fprintf(stderr, "gen-operations failed for %s: %v\n", skill.Name, genErr)
			continue
		}
		results = append(results, *result)
	}

	if jsonOutput {
		return writeJSON(stdout, results, stderr)
	}

	changed := 0
	for _, result := range results {
		status := "up to date"
		if result.Changed {
			changed++
			if check {
				status = "out of date"
			} else if result.Written {
				status = "updated"
			}
		}
		fmt.Fprintf(stdout, "%-32s %s  (%d commands, %s)\n", result.SkillName, result.SkillZid, result.CommandCount, status)
	}
	fmt.Fprintf(stdout, "\nGenerated: %d skill(s), %d changed; skipped %d without manifest", len(results), changed, skipped)
	if failed > 0 {
		fmt.Fprintf(stdout, "; %d failed", failed)
	}
	fmt.Fprintln(stdout)
	if check && changed > 0 {
		return 1
	}
	return 0
}

func printSkillsGenOperationsUsage(out io.Writer) {
	fmt.Fprintln(out, "# skm skills gen-operations")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Regenerate the SKILL.md Operations section from package.json, keeping the")
	fmt.Fprintln(out, "agent-facing documentation in sync with the commands `skills exec` can run.")
	fmt.Fprintln(out, "The section is wrapped in generator markers and replaced on every run.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  skm skills gen-operations <skill-zid> [--check] [--json]")
	fmt.Fprintln(out, "  skm skills gen-operations --all [--check] [--json]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  --check    exit 1 when SKILL.md would change, without writing")
	fmt.Fprintln(out, "  --json     structured result output")
	fmt.Fprintln(out, "  -h, --help help for skm skills gen-operations")
}

func writeJSONAndExit(stdout, stderr io.Writer, result *service.ExecResult) int {
	code := writeJSON(stdout, result, stderr)
	if code != 0 {
		return code
	}
	exitCode := result.ExitCode
	if exitCode < 0 {
		exitCode = 1
	}
	return exitCode
}

func printSkillsExecUsage(out io.Writer) {
	fmt.Fprintln(out, "# skm skills exec")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Run a command declared in the skill's package.json (scripts section).")
	fmt.Fprintln(out, "Only declared commands can be executed. Arguments after -- are appended")
	fmt.Fprintln(out, "to the command line (npm-run semantics). Linked copies resolve to their")
	fmt.Fprintln(out, "source directory, so execution always happens at the source of truth.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Before the first run in a working directory, managed dependency")
	fmt.Fprintln(out, "installation (opt-in via skm.runtime.deps) and the manifest's")
	fmt.Fprintln(out, "runtime.setup command run automatically (both idempotent; skm maintains")
	fmt.Fprintln(out, "the completion markers in ~/.skm/cache/exec/<zid>/).")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  skm skills exec <skill-zid> <command> [-- args...]")
	fmt.Fprintln(out, "  skm skills exec <skill-zid> --setup [--isolate] [--force]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  --input <json|@file|->   structured input; validated against the declared schema")
	fmt.Fprintln(out, "                           delivery (stdin/argv/env) follows the manifest")
	fmt.Fprintln(out, "  --env KEY=VAL            inject an environment variable (repeatable)")
	fmt.Fprintln(out, "  --yes                    required for commands that declare confirm")
	fmt.Fprintln(out, "  --timeout <sec>          override the manifest timeout")
	fmt.Fprintln(out, "  --isolate                run in a materialized cache copy")
	fmt.Fprintln(out, "                           (~/.skm/cache/exec/<zid>/), protecting the")
	fmt.Fprintln(out, "                           source directory from writes")
	fmt.Fprintln(out, "  --pin <hash>             run the version whose source hash starts with")
	fmt.Fprintln(out, "                           this value (8-64 hex); runs in a cache copy,")
	fmt.Fprintln(out, "                           never in the source directory. Discover hashes")
	fmt.Fprintln(out, "                           via `skm skills execs`")
	fmt.Fprintln(out, "  --setup                  run only runtime.setup and exit (idempotent)")
	fmt.Fprintln(out, "  --force                  with --setup: re-run even when already set up")
	fmt.Fprintln(out, "  --dry-run                print the resolved command without executing")
	fmt.Fprintln(out, "                           (bypasses the confirm gate, never runs)")
	fmt.Fprintln(out, "  --json                   structured result output")
	fmt.Fprintln(out, "  -h, --help               help for skm skills exec")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "The exit code is the executed command's exit code (124 on timeout).")
}

func runIssues(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("issues", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	view := fs.String("view", "latest", "issue view: latest or all")
	provider := fs.String("provider", "", "provider zid or name")
	severity := fs.String("severity", "", "filter by severity")
	code := fs.String("code", "", "filter by issue code")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	issues, err := deps.catalog.ListIssues(context.Background(), service.IssueListFilters{
		View:     *view,
		Provider: *provider,
		Severity: *severity,
		Code:     *code,
	})
	if err != nil {
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, issues, stderr)
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CREATED\tSEVERITY\tCODE\tPROVIDER\tPATH\tMESSAGE")
	for _, issue := range issues {
		providerName := ""
		if issue.Provider != nil {
			providerName = issue.Provider.Name
		}
		path := issue.RelativePath
		if path == "" {
			path = issue.RootPath
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			formatTime(issue.CreatedAt),
			issue.Severity,
			issue.Code,
			providerName,
			path,
			oneLine(issue.Message),
		)
	}
	_ = tw.Flush()
	fmt.Fprintf(stdout, "\nTotal issues: %d\n", len(issues))
	return 0
}

func runScan(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && isHelpToken(args[0]) {
		printScanUsage(stdout)
		return 0
	}

	if len(args) == 0 {
		fmt.Fprintln(stderr, "scan requires a subcommand: all or provider <zid>")
		return 2
	}

	switch args[0] {
	case "help":
		printScanUsage(stdout)
		return 0
	case "all":
		return runScanAll(args[1:], stdout, stderr)
	case "provider":
		return runScanProvider(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown scan subcommand: %s\n", args[0])
		return 2
	}
}

func runScanAll(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("scan all", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	result, err := deps.scan.ScanAllProviders(context.Background())
	if err != nil {
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, result, stderr)
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "JOB\tSTATUS\tADDED\tREMOVED\tCHANGED\tINVALID\tCONFLICTS")
	for _, job := range result.Jobs {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
			job.Zid,
			job.Status,
			job.AddedCount,
			job.RemovedCount,
			job.ChangedCount,
			job.InvalidCount,
			job.ConflictCount,
		)
	}
	_ = tw.Flush()
	fmt.Fprintf(stdout, "\nCompleted %d scan jobs\n", len(result.Jobs))
	return 0
}

func runScanProvider(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("scan provider", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: skm scan provider <provider-zid>")
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	job, err := deps.scan.ScanProviderByZid(context.Background(), fs.Arg(0))
	if err != nil {
		if errors.Is(err, service.ErrProviderNotFound) {
			fmt.Fprintf(stderr, "provider not found: %s\n", fs.Arg(0))
			return 1
		}
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, job, stderr)
	}

	fmt.Fprintf(stdout, "Job: %s\n", job.Zid)
	fmt.Fprintf(stdout, "Status: %s\n", job.Status)
	fmt.Fprintf(stdout, "Started: %s\n", formatTime(job.StartedAt))
	fmt.Fprintf(stdout, "Finished: %s\n", formatOptionalTime(job.FinishedAt))
	fmt.Fprintf(stdout, "Added: %d  Removed: %d  Changed: %d  Invalid: %d  Conflicts: %d\n",
		job.AddedCount,
		job.RemovedCount,
		job.ChangedCount,
		job.InvalidCount,
		job.ConflictCount,
	)
	if len(job.LogLines) > 0 {
		fmt.Fprintln(stdout, "Logs:")
		for _, line := range job.LogLines {
			fmt.Fprintf(stdout, "- %s\n", line)
		}
	}
	return 0
}

type cliDeps struct {
	cfg     *config.Config
	db      *gorm.DB
	catalog *service.CatalogService
	scan    *service.ScanService
	exec    *service.ExecService
}

func openDeps() (*cliDeps, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	gdb, err := dbpkg.Open(dbpkg.Config{
		Driver:  cfg.DBDriver,
		DSN:     cfg.DBDSN,
		LogMode: "silent",
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return &cliDeps{
		cfg:     cfg,
		db:      gdb,
		catalog: service.NewCatalogService(gdb),
		scan:    service.NewScanService(gdb),
		exec:    service.NewExecService(gdb),
	}, nil
}

func (d *cliDeps) close() {
	if d == nil || d.db == nil {
		return
	}
	sqlDB, err := d.db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func isHelpToken(value string) bool {
	return value == "help" || value == "--help" || value == "-h"
}

func writeJSON(stdout io.Writer, value any, stderr io.Writer) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return printError(stderr, err)
	}
	return 0
}

func printError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "error: %v\n", err)
	return 1
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "# skm")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "skm is a CLI tool that helps you manage skill providers, skills, scans, and desktop workflows.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  skm [command]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Available Commands:")
	fmt.Fprintln(out, "  dashboard   Show dashboard summary")
	fmt.Fprintln(out, "  help        Help about any command")
	fmt.Fprintln(out, "  issues      List catalog issues")
	fmt.Fprintln(out, "  providers   Manage providers: list, add, update, delete")
	fmt.Fprintln(out, "  scan        Run provider scans")
	fmt.Fprintln(out, "  skills      Manage skills: list, get, to, delete, link, move, sync, exec, execs")
	fmt.Fprintln(out, "  version     Print the current version")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  -h, --help   help for skm")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Use \"skm [command] --help\" for more information about a command.")
}

func printProvidersUsage(out io.Writer) {
	fmt.Fprintln(out, "# skm providers")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "skm providers lets you list, create, update, and delete providers.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  skm providers [--json]")
	fmt.Fprintln(out, "  skm providers add --name <name> --type <type> --root <path> [--scan-mode recursive|shallow] [--enabled=true] [--priority 100] [--icon name] [--description text] [--json]")
	fmt.Fprintln(out, "  skm providers update <provider-zid> [--name <name>] [--type <type>] [--root <path>] [--scan-mode recursive|shallow] [--enabled true|false] [--priority 100] [--icon name] [--description text] [--json]")
	fmt.Fprintln(out, "  skm providers delete <provider-zid>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Available Commands:")
	fmt.Fprintln(out, "  add       Create a provider")
	fmt.Fprintln(out, "  delete    Delete a provider")
	fmt.Fprintln(out, "  update    Update a provider")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  -h, --help   help for skm providers")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Use \"skm providers [command] --help\" for more information about a command.")
}

func printSkillsUsage(out io.Writer) {
	fmt.Fprintln(out, "# skm skills")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "skm skills lets you list skills, inspect details, manage .to metadata, and move or sync skill copies.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  skm skills [-q|--query text] [--provider zid-or-name] [--category value] [--tag value] [--status value] [--sort name|provider|status|lastScanned] [--conflict true|false] [--digest] [--json]")
	fmt.Fprintln(out, "  skm skills get <skill-zid> [path] [--files] [--commands] [--json]")
	fmt.Fprintln(out, "  skm skills to [--provider-path <path>] [--directory <path> ...] [--include <pattern> ...] [--exclude <pattern> ...] [--json]")
	fmt.Fprintln(out, "  skm skills delete <skill-zid> [--force] [--json]")
	fmt.Fprintln(out, "  skm skills link <skill-zid> --to <provider-zid> [--json]")
	fmt.Fprintln(out, "  skm skills move <skill-zid> --to <provider-zid> [--json]")
	fmt.Fprintln(out, "  skm skills sync <skill-zid> [--json]")
	fmt.Fprintln(out, "  skm skills sync-copies <skill-zid> [--json]")
	fmt.Fprintln(out, "  skm skills exec <skill-zid> <command> [--input <json|@file|->] [--env KEY=VAL ...] [--yes] [--timeout sec] [--isolate] [--pin hash] [--dry-run] [--json] [-- args...]")
	fmt.Fprintln(out, "  skm skills exec <skill-zid> --setup [--isolate] [--force] [--pin hash] [--dry-run] [--json]")
	fmt.Fprintln(out, "  skm skills execs [--skill <skill-zid>] [--limit N] [--json]")
	fmt.Fprintln(out, "  skm skills gen-operations <skill-zid>|--all [--check] [--json]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Available Commands:")
	fmt.Fprintln(out, "  delete    Delete a skill")
	fmt.Fprintln(out, "  exec      Run a command declared in the skill's package.json")
	fmt.Fprintln(out, "  execs     List audited exec invocations (history, pins, outcomes)")
	fmt.Fprintln(out, "  gen-operations  Regenerate the SKILL.md Operations section from package.json")
	fmt.Fprintln(out, "  get       Show skill details, SKILL.md content, a file tree, or one file")
	fmt.Fprintln(out, "  link      Create an attached copy in another provider")
	fmt.Fprintln(out, "  move      Move a skill to another provider")
	fmt.Fprintln(out, "  sync      Sync an attached copy from its source")
	fmt.Fprintln(out, "  sync-copies  Sync all attached copies from a relation source")
	fmt.Fprintln(out, "  to        Create or update .to metadata in the current skill directory")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  -q, --query string   fuzzy search skills by name, slug, tags, category, provider, or summary")
	fmt.Fprintln(out, "                       multiple space-separated words must all match; results are ranked by relevance")
	fmt.Fprintln(out, "                       combines with --provider, --category, --tag, --status, and --conflict")
	fmt.Fprintln(out, "      --digest         compact one-line-per-skill digest (NAME/ZID/PROVIDER/STATUS/COMMANDS/SUMMARY)")
	fmt.Fprintln(out, "                       for LLM context injection; combines with all list filters")
	fmt.Fprintln(out, "  -h, --help           help for skm skills")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Use \"skm skills [command] --help\" for more information about a command.")
}

func printScanUsage(out io.Writer) {
	fmt.Fprintln(out, "# skm scan")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "skm scan runs catalog scans across all providers or a single provider.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  skm scan all [--json]")
	fmt.Fprintln(out, "  skm scan provider <provider-zid> [--json]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Available Commands:")
	fmt.Fprintln(out, "  all       Run scans for all providers")
	fmt.Fprintln(out, "  provider  Run a scan for one provider")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  -h, --help   help for skm scan")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Use \"skm scan [command] --help\" for more information about a command.")
}

func printProviderDetails(stdout io.Writer, provider *models.Provider) {
	fmt.Fprintf(stdout, "Provider: %s (%s)\n", provider.Name, provider.Zid)
	fmt.Fprintf(stdout, "Type: %s\n", provider.Type)
	fmt.Fprintf(stdout, "Root: %s\n", provider.RootPath)
	fmt.Fprintf(stdout, "Enabled: %t\n", provider.Enabled)
	fmt.Fprintf(stdout, "Priority: %d\n", provider.Priority)
	fmt.Fprintf(stdout, "Scan mode: %s\n", provider.ScanMode)
	if provider.Icon != "" {
		fmt.Fprintf(stdout, "Icon: %s\n", provider.Icon)
	}
	if provider.Description != "" {
		fmt.Fprintf(stdout, "Description: %s\n", provider.Description)
	}
	if provider.LastScanStatus != "" {
		fmt.Fprintf(stdout, "Last scan status: %s\n", provider.LastScanStatus)
	}
	if provider.LastScannedAt != nil {
		fmt.Fprintf(stdout, "Last scanned: %s\n", formatTime(*provider.LastScannedAt))
	}
}

func printSkillDetails(stdout io.Writer, skill *models.Skill) {
	fmt.Fprintf(stdout, "Skill: %s (%s)\n", skill.Name, skill.Zid)
	fmt.Fprintf(stdout, "Provider: %s (%s)\n", skill.Provider.Name, skill.Provider.Zid)
	fmt.Fprintf(stdout, "Root: %s\n", skill.RootPath)
	fmt.Fprintf(stdout, "Directory: %s\n", skill.DirectoryName)
	fmt.Fprintf(stdout, "Status: %s\n", skill.Status)
	fmt.Fprintf(stdout, "Conflict: %t\n", skill.IsConflict)
	if skill.Category != "" {
		fmt.Fprintf(stdout, "Category: %s\n", skill.Category)
	}
	if len(skill.Tags) > 0 {
		fmt.Fprintf(stdout, "Tags: %s\n", strings.Join(skill.Tags, ", "))
	}
	if skill.Summary != "" {
		fmt.Fprintf(stdout, "Summary: %s\n", oneLine(skill.Summary))
	}
	if len(skill.Commands) > 0 {
		fmt.Fprintf(stdout, "Commands: %d declared (see --commands)\n", len(skill.Commands))
	}
	if skill.Relation != nil {
		fmt.Fprintf(stdout, "Relation mode: %s\n", skill.Relation.Mode)
		if skill.Relation.FromPath != "" {
			fmt.Fprintf(stdout, "Relation from: %s\n", skill.Relation.FromPath)
		}
		if len(skill.Relation.Directories) > 0 {
			fmt.Fprintf(stdout, "Relation targets: %s\n", strings.Join(skill.Relation.Directories, ", "))
		}
	}
}

func formatTime(value time.Time) string {
	return value.Local().Format(time.RFC3339)
}

func parseSinglePositional(fs *flag.FlagSet, args []string) (string, error) {
	positionals, err := parsePositionals(fs, args, 1)
	if err != nil {
		return "", err
	}
	if len(positionals) == 0 {
		return "", nil
	}
	return positionals[0], nil
}

// parsePositionals accepts up to max positional arguments mixed with flags in
// any order and returns them in order.
func parsePositionals(fs *flag.FlagSet, args []string, max int) ([]string, error) {
	var positionals []string
	parseArgs := args
	for len(parseArgs) > 0 && len(positionals) < max && !strings.HasPrefix(parseArgs[0], "-") {
		positionals = append(positionals, parseArgs[0])
		parseArgs = parseArgs[1:]
	}
	if err := fs.Parse(parseArgs); err != nil {
		return nil, err
	}
	extra := 0
	for index := 0; index < fs.NArg(); index++ {
		if len(positionals) < max {
			positionals = append(positionals, fs.Arg(index))
			continue
		}
		extra++
	}
	if extra > 0 {
		return nil, fmt.Errorf("unexpected extra arguments")
	}
	return positionals, nil
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}

func oneLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	return value
}

// repeatStringFlag collects each occurrence of a flag verbatim (no splitting),
// for values that may legally contain commas, such as KEY=VAL pairs.
type repeatStringFlag struct {
	values []string
}

func (f *repeatStringFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *repeatStringFlag) Set(value string) error {
	f.values = append(f.values, value)
	return nil
}

func (f *repeatStringFlag) Values() []string {
	return append([]string{}, f.values...)
}

type multiStringFlag struct {
	values []string
}

func (f *multiStringFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *multiStringFlag) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			f.values = append(f.values, trimmed)
		}
	}
	return nil
}

func (f *multiStringFlag) Values() []string {
	return append([]string{}, f.values...)
}
