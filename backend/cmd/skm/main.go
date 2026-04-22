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
		return 0
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
	default:
		fmt.Fprintf(stderr, "unknown skills subcommand: %s\n", args[0])
		return 2
	}
}

func runSkillsList(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("skills", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	query := fs.String("query", "", "search by name or summary")
	provider := fs.String("provider", "", "provider zid or name")
	category := fs.String("category", "", "filter by category")
	tag := fs.String("tag", "", "filter by tag")
	status := fs.String("status", "", "filter by status")
	sortBy := fs.String("sort", "name", "sort by name, provider, status, lastScanned")
	conflict := fs.String("conflict", "", "filter by conflict: true or false")
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

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	skills, err := deps.catalog.ListSkills(context.Background(), service.SkillListFilters{
		Query:    *query,
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
	fs := newFlagSet("skills get", stderr)
	jsonOutput := fs.Bool("json", false, "output JSON")
	skillZid, err := parseSinglePositional(fs, args)
	if err != nil {
		return 2
	}
	if skillZid == "" {
		fmt.Fprintln(stderr, "usage: skm skills get <skill-zid>")
		return 2
	}

	deps, err := openDeps()
	if err != nil {
		return printError(stderr, err)
	}
	defer deps.close()

	skill, err := deps.catalog.GetSkill(context.Background(), skillZid)
	if err != nil {
		if errors.Is(err, service.ErrSkillNotFound) {
			fmt.Fprintf(stderr, "skill not found: %s\n", skillZid)
			return 1
		}
		return printError(stderr, err)
	}

	if *jsonOutput {
		return writeJSON(stdout, skill, stderr)
	}

	printSkillDetails(stdout, skill)
	return 0
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
	fmt.Fprintln(out, "  skills      Manage skills: list, get, to, delete, link, move, sync")
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
	fmt.Fprintln(out, "  skm skills [--query text] [--provider zid-or-name] [--category value] [--tag value] [--status value] [--sort name|provider|status|lastScanned] [--conflict true|false] [--json]")
	fmt.Fprintln(out, "  skm skills get <skill-zid> [--json]")
	fmt.Fprintln(out, "  skm skills to [--provider-path <path>] [--directory <path> ...] [--include <pattern> ...] [--exclude <pattern> ...] [--json]")
	fmt.Fprintln(out, "  skm skills delete <skill-zid> [--force] [--json]")
	fmt.Fprintln(out, "  skm skills link <skill-zid> --to <provider-zid> [--json]")
	fmt.Fprintln(out, "  skm skills move <skill-zid> --to <provider-zid> [--json]")
	fmt.Fprintln(out, "  skm skills sync <skill-zid> [--json]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Available Commands:")
	fmt.Fprintln(out, "  delete    Delete a skill")
	fmt.Fprintln(out, "  get       Show a single skill")
	fmt.Fprintln(out, "  link      Create an attached copy in another provider")
	fmt.Fprintln(out, "  move      Move a skill to another provider")
	fmt.Fprintln(out, "  sync      Sync an attached copy from its source")
	fmt.Fprintln(out, "  to        Create or update .to metadata in the current skill directory")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  -h, --help   help for skm skills")
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
	positional := ""
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		positional = args[0]
		parseArgs = args[1:]
	}
	if err := fs.Parse(parseArgs); err != nil {
		return "", err
	}
	if positional == "" && fs.NArg() > 0 {
		positional = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		return "", fmt.Errorf("unexpected extra arguments")
	}
	return positional, nil
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
