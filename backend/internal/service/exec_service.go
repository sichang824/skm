package service

import (
	"backend-go/internal/models"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"gorm.io/gorm"
)

// ExecService runs skill commands declared in package.json. Only commands
// registered in the manifest scripts section can be executed (allowlist).
type ExecService struct {
	catalog *CatalogService
	// db records exec audit entries (best effort; never fails an exec).
	db *gorm.DB
	// cacheRoot holds materialized cache copies and setup markers, by
	// default ~/.skm/cache/exec/<zid>/. Tests override it.
	cacheRoot string
}

func NewExecService(db *gorm.DB) *ExecService {
	return &ExecService{catalog: NewCatalogService(db), db: db, cacheRoot: defaultExecCacheRoot()}
}

// ErrExecManifestMissing means the skill directory has no package.json.
var ErrExecManifestMissing = errors.New("skill declares no executable manifest (missing package.json)")

// ExecCommandNotFound means the requested command is not in scripts.
type ExecCommandNotFound struct {
	Command   string
	Available []string
}

func (e *ExecCommandNotFound) Error() string {
	if len(e.Available) == 0 {
		return fmt.Sprintf("command %q not found: manifest declares no commands", e.Command)
	}
	return fmt.Sprintf("command %q not found; available: %s", e.Command, strings.Join(e.Available, ", "))
}

// ExecMissingEnv lists required environment variables that are not set.
type ExecMissingEnv struct {
	Missing []string
}

func (e *ExecMissingEnv) Error() string {
	return fmt.Sprintf("missing required environment variables: %s", strings.Join(e.Missing, ", "))
}

// ExecConfirmRequired blocks a confirm-gated command until --yes is passed.
type ExecConfirmRequired struct {
	Command string
	Message string
}

func (e *ExecConfirmRequired) Error() string {
	return fmt.Sprintf("command %q requires confirmation: %s (re-run with --yes after the user agrees)", e.Command, e.Message)
}

// ExecInputInvalid reports structured input that fails the declared schema,
// or input provided to a command that declares none.
type ExecInputInvalid struct {
	Reason string
}

func (e *ExecInputInvalid) Error() string {
	return e.Reason
}

// ExecRootMissing means no executable directory exists on disk for the skill.
type ExecRootMissing struct {
	Path string
}

func (e *ExecRootMissing) Error() string {
	return fmt.Sprintf("skill directory does not exist on disk: %s", e.Path)
}

// ErrExecSetupMissing means the manifest declares no skm.runtime.setup.
var ErrExecSetupMissing = errors.New("skill declares no runtime.setup command")

// ExecPinInvalid means --pin is not 8-64 lowercase hex characters.
type ExecPinInvalid struct {
	Pin string
}

func (e *ExecPinInvalid) Error() string {
	return fmt.Sprintf("invalid --pin %q: expected 8-64 hex characters of a source hash", e.Pin)
}

// ExecPinUnavailable means neither the cache copy nor the current source
// tree matches the requested pin, so the pinned version cannot be recovered.
type ExecPinUnavailable struct {
	SkillZid string
	Pin      string
}

func (e *ExecPinUnavailable) Error() string {
	return fmt.Sprintf("no executable version matches pin %q for skill %s; run `skm skills execs --skill %s` to see source hashes of previous runs", e.Pin, e.SkillZid, e.SkillZid)
}

var execPinPattern = regexp.MustCompile(`^[0-9a-f]{8,64}$`)

// validateExecPin checks pin format. An empty pin is valid (no pin).
// Callers lowercase the pin first; matching is prefix-based.
func validateExecPin(pin string) error {
	if pin == "" {
		return nil
	}
	if !execPinPattern.MatchString(pin) {
		return &ExecPinInvalid{Pin: pin}
	}
	return nil
}

// ExecRequest describes one exec invocation.
type ExecRequest struct {
	SkillZid        string
	Command         string
	Args            []string // positional arguments appended after the command line
	InputJSON       []byte   // structured input (raw JSON bytes)
	Env             []string // KEY=VAL injections
	AssumeYes       bool
	TimeoutOverride time.Duration
	DryRun          bool
	Isolate         bool   // force execution in a materialized cache copy
	Pin             string // run the version whose source hash starts with this (normalized lowercase hex)
	Trigger         string // "cli" (default) or "http"; recorded by the audit trail
	Stdout          io.Writer // nil -> captured into ExecResult.Stdout
	Stderr          io.Writer // nil -> captured into ExecResult.Stderr
}

// ExecPlan is the fully resolved execution, produced by --dry-run and
// included in results for observability.
type ExecPlan struct {
	WorkDir        string   `json:"workDir"`
	Mode           string   `json:"mode"` // source | cache
	SourceDir      string   `json:"sourceDir,omitempty"`
	CacheReused    bool     `json:"cacheReused,omitempty"`
	Materialized   bool     `json:"materialized,omitempty"`
	Pin            string   `json:"pin,omitempty"`
	CommandLine    string   `json:"commandLine"`
	Args           []string `json:"args,omitempty"`
	InputVia       string   `json:"inputVia,omitempty"`
	InputBytes     int      `json:"inputBytes,omitempty"`
	EnvAdditions   []string `json:"envAdditions,omitempty"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"`
	Confirm        string   `json:"confirm,omitempty"`
	Setup          string   `json:"setup,omitempty"`
	SetupSkipped   bool     `json:"setupSkipped,omitempty"`
	Deps           []string `json:"deps,omitempty"`
	DepsSkipped    bool     `json:"depsSkipped,omitempty"`
}

// SetupInfo describes what happened with skm.runtime.setup during one
// invocation: it ran (Ran), was skipped because the completion marker is
// still fresh (Skipped), or failed (non-zero ExitCode / TimedOut).
type SetupInfo struct {
	Command    string `json:"command"`
	Ran        bool   `json:"ran,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
	ExitCode   int    `json:"exitCode,omitempty"`
	TimedOut   bool   `json:"timedOut,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	// Output carries the captured setup output when it was not streamed to
	// the caller's writers (e.g. --json mode).
	Output string `json:"output,omitempty"`
}

// ExecResult is the outcome of one exec invocation.
type ExecResult struct {
	OK         bool       `json:"ok"`
	ExitCode   int        `json:"exitCode"`
	TimedOut   bool       `json:"timedOut"`
	DryRun     bool       `json:"dryRun,omitempty"`
	Aborted    string     `json:"aborted,omitempty"` // e.g. "setup-failed"
	SkillZid   string     `json:"skillZid"`
	SkillName  string     `json:"skillName"`
	Command    string     `json:"command"`
	WorkDir    string     `json:"workDir"`
	DurationMs int64      `json:"durationMs"`
	Stdout     string     `json:"stdout,omitempty"`
	Stderr     string     `json:"stderr,omitempty"`
	Deps       *DepsInfo  `json:"deps,omitempty"`
	Setup      *SetupInfo `json:"setup,omitempty"`
	Plan       *ExecPlan  `json:"plan,omitempty"`
}

// Exec resolves, pre-checks, and runs one declared command.
//
// Working-directory resolution follows the design doc §4: the source
// directory wins (linked copies resolve to their source); --isolate forces a
// materialized cache copy; when no local directory exists, a still-fresh
// cache copy is the only fallback. Before the first real run, the manifest's
// skm.runtime.setup command executes automatically (idempotent).
func (s *ExecService) Exec(ctx context.Context, req *ExecRequest) (result *ExecResult, retErr error) {
	pin := strings.ToLower(strings.TrimSpace(req.Pin))

	// Audit trail: every non-dry-run invocation is recorded exactly once,
	// covering every return path below (pre-check rejections included).
	record := s.newExecRecord(req.DryRun, req.SkillZid, req.Command, req.Trigger, req.Args, req.Env, pin)
	if record != nil {
		defer func() { s.finalizeExecRecord(ctx, record, result, retErr) }()
	}

	if err := validateExecPin(pin); err != nil {
		return nil, err
	}

	skill, err := s.catalog.GetSkill(ctx, req.SkillZid)
	if err != nil {
		return nil, err
	}
	if record != nil {
		record.SkillName = skill.Name
	}

	sourceDir := resolveExecDir(skill)
	loc, err := s.resolveExecLocation(skill, sourceDir, isExistingDir(sourceDir), req.Isolate, req.DryRun, pin)
	if err != nil {
		return nil, err
	}
	if record != nil {
		record.WorkDir = loc.WorkDir
		record.Mode = loc.Mode
		record.SourceHash = s.sourceHashForAudit(loc, sourceDir)
	}

	manifest, err := LoadManifest(loc.ManifestDir)
	if err != nil {
		if errors.Is(err, ErrManifestNotFound) {
			return nil, ErrExecManifestMissing
		}
		return nil, err
	}

	commandLine, declared := manifest.Scripts[req.Command]
	if !declared {
		available := make([]string, 0, len(manifest.Scripts))
		for name := range manifest.Scripts {
			available = append(available, name)
		}
		sort.Strings(available)
		return nil, &ExecCommandNotFound{Command: req.Command, Available: available}
	}
	annotation := findCommandAnnotation(manifest.Commands, req.Command)

	// Pre-check: structured input.
	inputVia := ""
	if len(req.InputJSON) > 0 {
		if annotation == nil || annotation.InputVia == "" {
			return nil, &ExecInputInvalid{Reason: fmt.Sprintf("command %q declares no structured input; pass arguments after -- instead", req.Command)}
		}
		inputVia = annotation.InputVia
		if schema, hasSchema := manifest.InputSchemas[req.Command]; hasSchema {
			if err := validateJSON(schema, req.InputJSON); err != nil {
				return nil, &ExecInputInvalid{Reason: fmt.Sprintf("input for command %q is invalid: %v", req.Command, err)}
			}
		}
	} else if annotation != nil && annotation.InputVia != "" {
		inputVia = annotation.InputVia
	}
	if record != nil {
		record.InputVia = inputVia
	}

	// Pre-check: confirm gate. Dry runs never execute, so they preview
	// confirm-gated commands without --yes.
	if annotation != nil && annotation.Confirm != "" && !req.AssumeYes && !req.DryRun {
		return nil, &ExecConfirmRequired{Command: req.Command, Message: annotation.Confirm}
	}

	// Pre-check: required env.
	envAdditions := append([]string{}, req.Env...)
	requiredEnv := append(append([]string{}, manifest.RuntimeEnv...), envNames(annotation)...)
	if missing := missingEnvNames(requiredEnv, envAdditions); len(missing) > 0 {
		return nil, &ExecMissingEnv{Missing: missing}
	}

	// Resolve timeout: override > manifest annotation > none.
	timeout := req.TimeoutOverride
	if timeout <= 0 && annotation != nil && annotation.TimeoutSeconds > 0 {
		timeout = time.Duration(annotation.TimeoutSeconds) * time.Second
	}

	// Assemble positional arguments: argv-mode input first, then -- args.
	runArgs := make([]string, 0, len(req.Args)+1)
	if inputVia == "argv" && len(req.InputJSON) > 0 {
		runArgs = append(runArgs, string(req.InputJSON))
	}
	runArgs = append(runArgs, req.Args...)

	// npm-run semantics: arguments are shell-quoted and appended to the end
	// of the declared command line. Quoting (not interpolation into an
	// unquoted template) is what keeps this injection-safe.
	fullLine := commandLine
	if len(runArgs) > 0 {
		quoted := make([]string, 0, len(runArgs))
		for _, arg := range runArgs {
			quoted = append(quoted, shellQuote(arg))
		}
		fullLine = commandLine + " " + strings.Join(quoted, " ")
	}

	plan := &ExecPlan{
		WorkDir:        loc.WorkDir,
		Mode:           loc.Mode,
		CacheReused:    loc.CacheReused,
		Materialized:   loc.Materialized,
		Pin:            pin,
		CommandLine:    fullLine,
		Args:           runArgs,
		InputVia:       inputVia,
		InputBytes:     len(req.InputJSON),
		EnvAdditions:   envAdditions,
		TimeoutSeconds: int(timeout / time.Second),
	}
	if loc.Mode == "cache" {
		plan.SourceDir = sourceDir
	}
	if annotation != nil {
		plan.Confirm = annotation.Confirm
	}
	if manifest.Deps.Declared {
		plan.Deps, plan.DepsSkipped = s.depsPlan(manifest, s.cacheDirFor(skill.Zid), loc.WorkDir)
	}
	if manifest.SetupCommand != "" {
		plan.Setup = manifest.SetupCommand
		entry := readSetupEntry(s.cacheDirFor(skill.Zid), loc.WorkDir)
		plan.SetupSkipped = setupEntryFresh(entry, manifest.Hash())
	}

	result = &ExecResult{
		SkillZid:  skill.Zid,
		SkillName: skill.Name,
		Command:   req.Command,
		WorkDir:   loc.WorkDir,
		Plan:      plan,
	}
	if req.DryRun {
		result.DryRun = true
		result.OK = true
		return result, nil
	}

	// Managed dependency installation (opt-in via skm.runtime.deps) runs
	// before setup, which may rely on installed packages. A failed install
	// aborts the command exactly like a failed setup: the script never
	// starts.
	depsInfo, depsErr := s.ensureDeps(ctx, skill, manifest, loc.WorkDir, req.Stdout, req.Stderr)
	if depsErr != nil {
		return nil, depsErr
	}
	if depsInfo != nil {
		result.Deps = depsInfo
		if depsInfo.ExitCode != 0 || depsInfo.TimedOut {
			result.Aborted = "deps-failed"
			result.ExitCode = depsInfo.ExitCode
			result.TimedOut = depsInfo.TimedOut
			result.DurationMs = depsInfo.DurationMs
			return result, nil
		}
	}

	// Automatic one-time setup before the first real run in this working
	// directory. A failed setup aborts the command; the script never starts.
	if manifest.SetupCommand != "" {
		info, setupErr := s.ensureSetup(ctx, skill, manifest, loc.WorkDir, req.Stdout, req.Stderr, false)
		if setupErr != nil {
			return nil, setupErr
		}
		result.Setup = info
		if info.ExitCode != 0 || info.TimedOut {
			result.Aborted = "setup-failed"
			result.ExitCode = info.ExitCode
			result.TimedOut = info.TimedOut
			result.DurationMs = info.DurationMs
			return result, nil
		}
	}

	env := buildExecEnv(loc.WorkDir, skill, manifest, req.Command, req.Env, inputVia, req.InputJSON)

	started := time.Now()
	runCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(runCtx, "sh", "-c", fullLine)
	cmd.Dir = loc.WorkDir
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			// Kill the whole process group so children of the shell die too.
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	cmd.WaitDelay = 5 * time.Second

	var stdoutBuffer, stderrBuffer bytes.Buffer
	if req.Stdout != nil {
		cmd.Stdout = req.Stdout
	} else {
		cmd.Stdout = &stdoutBuffer
	}
	if req.Stderr != nil {
		cmd.Stderr = req.Stderr
	} else {
		cmd.Stderr = &stderrBuffer
	}
	if inputVia == "stdin" && len(req.InputJSON) > 0 {
		cmd.Stdin = bytes.NewReader(req.InputJSON)
	}

	runErr := cmd.Run()
	result.DurationMs = time.Since(started).Milliseconds()
	result.Stdout = stdoutBuffer.String()
	result.Stderr = stderrBuffer.String()

	if runCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.ExitCode = 124
		return result, nil
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return nil, fmt.Errorf("start command %q: %w", req.Command, runErr)
	}
	result.OK = true
	return result, nil
}

func resolveExecDir(skill *models.Skill) string {
	if skill.Relation != nil && skill.Relation.Mode == "from" {
		sourcePath := strings.TrimSpace(skill.Relation.FromPath)
		if sourcePath != "" {
			if info, err := os.Stat(sourcePath); err == nil && info.IsDir() {
				return sourcePath
			}
		}
	}
	return skill.RootPath
}

func findCommandAnnotation(commands []models.SkillCommand, name string) *models.SkillCommand {
	for index := range commands {
		if commands[index].Name == name {
			return &commands[index]
		}
	}
	return nil
}

func envNames(annotation *models.SkillCommand) []string {
	if annotation == nil {
		return nil
	}
	return annotation.Env
}

// missingEnvNames returns required names that are neither in the ambient
// environment nor supplied via --env. skm never loads .env files itself.
func missingEnvNames(required, additions []string) []string {
	provided := map[string]struct{}{}
	for _, entry := range os.Environ() {
		if name, _, found := strings.Cut(entry, "="); found {
			provided[name] = struct{}{}
		}
	}
	for _, entry := range additions {
		if name, _, found := strings.Cut(entry, "="); found {
			provided[name] = struct{}{}
		}
	}
	missing := make([]string, 0)
	seen := map[string]struct{}{}
	for _, name := range required {
		if _, ok := provided[name]; ok {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		missing = append(missing, name)
	}
	sort.Strings(missing)
	return missing
}

func buildExecEnv(workDir string, skill *models.Skill, manifest *ParsedManifest, commandName string, envAdditions []string, inputVia string, inputJSON []byte) []string {
	env := append([]string{}, os.Environ()...)
	env = append(env, envAdditions...)
	env = append(env,
		"SKM_SKILL_ROOT="+workDir,
		"SKM_SKILL_ZID="+skill.Zid,
		"SKM_SKILL_NAME="+skill.Name,
		"SKM_SKILL_VERSION="+manifest.Version,
		"SKM_COMMAND="+commandName,
	)
	if inputVia == "env" && len(inputJSON) > 0 {
		env = append(env, "SKM_INPUT="+string(inputJSON))
	}
	return injectManagedPath(env, workDir)
}

// injectManagedPath prepends the skill-local bin directories to PATH when
// they exist: .venv/bin for managed python installs and node_modules/.bin
// for node. Commands run via sh -c rather than npm run, so node_modules/.bin
// must be on PATH for parity with npm-run semantics. Go applies the last
// value of a duplicate env key, so the existing PATH entry is replaced, not
// duplicated.
func injectManagedPath(env []string, workDir string) []string {
	bins := make([]string, 0, 2)
	if venvBin := filepath.Join(workDir, ".venv", "bin"); isExistingDir(venvBin) {
		bins = append(bins, venvBin)
	}
	if npmBin := filepath.Join(workDir, "node_modules", ".bin"); isExistingDir(npmBin) {
		bins = append(bins, npmBin)
	}
	if len(bins) == 0 {
		return env
	}
	for index := len(env) - 1; index >= 0; index-- {
		if current, ok := strings.CutPrefix(env[index], "PATH="); ok {
			env[index] = "PATH=" + strings.Join(append(bins, current), string(os.PathListSeparator))
			return env
		}
	}
	return append(env, "PATH="+strings.Join(bins, string(os.PathListSeparator)))
}

// CommandsView lists the executable commands of one skill, for discovery by
// agents (CLI `skills get --commands`) and the App UI.
type CommandsView struct {
	SkillZid   string                `json:"skillZid"`
	SkillName  string                `json:"skillName"`
	SkillRoot  string                `json:"skillRoot"`
	Source     string                `json:"source"` // disk | catalog
	RuntimeEnv []string              `json:"runtimeEnv,omitempty"`
	Setup      string                `json:"setup,omitempty"`
	Deps       *ManifestDeps         `json:"deps,omitempty"`
	Commands   []models.SkillCommand `json:"commands"`
	Note       string                `json:"note,omitempty"`
}

// Commands lists the executable commands declared in the skill's
// package.json. It prefers a live parse from disk (freshest, resolving
// linked copies to their source) and falls back to the catalog snapshot
// when the manifest cannot be read.
func (s *ExecService) Commands(ctx context.Context, zid string) (*CommandsView, error) {
	skill, err := s.catalog.GetSkill(ctx, zid)
	if err != nil {
		return nil, err
	}
	root := resolveExecDir(skill)
	view := &CommandsView{
		SkillZid:  skill.Zid,
		SkillName: skill.Name,
		SkillRoot: root,
		Source:    "disk",
	}
	manifest, err := LoadManifest(root)
	switch {
	case err == nil:
		view.RuntimeEnv = manifest.RuntimeEnv
		view.Setup = manifest.SetupCommand
		view.Commands = manifest.Commands
		if manifest.Deps.Declared {
			deps := manifest.Deps
			view.Deps = &deps
		}
	case errors.Is(err, ErrManifestNotFound):
		view.Source = "catalog"
		view.Commands = skill.Commands
		if len(view.Commands) == 0 {
			view.Note = "skill declares no executable commands (no package.json)"
		}
	default:
		view.Source = "catalog"
		view.Commands = skill.Commands
		view.Note = fmt.Sprintf("package.json is invalid (%v); showing catalog snapshot", err)
	}
	return view, nil
}

// SetupRequest describes an explicit `skills exec --setup` invocation.
type SetupRequest struct {
	SkillZid string
	Isolate  bool // prepare the cache copy instead of the source directory
	Force    bool // re-run setup even when the completion marker is fresh
	DryRun   bool
	Pin      string // prepare the pinned cache copy instead of the latest content
	Trigger  string // "cli" (default) or "http"; recorded by the audit trail
	Stdout   io.Writer
	Stderr   io.Writer
}

// RunSetup executes the manifest's skm.runtime.setup command on its own
// (idempotent: the completion marker is maintained by skm). Managed
// dependency installation runs first because setup may need it.
func (s *ExecService) RunSetup(ctx context.Context, req *SetupRequest) (result *ExecResult, retErr error) {
	pin := strings.ToLower(strings.TrimSpace(req.Pin))

	record := s.newExecRecord(req.DryRun, req.SkillZid, setupSentinelCommand, req.Trigger, nil, nil, pin)
	if record != nil {
		defer func() { s.finalizeExecRecord(ctx, record, result, retErr) }()
	}

	if err := validateExecPin(pin); err != nil {
		return nil, err
	}

	skill, err := s.catalog.GetSkill(ctx, req.SkillZid)
	if err != nil {
		return nil, err
	}
	if record != nil {
		record.SkillName = skill.Name
	}

	sourceDir := resolveExecDir(skill)
	loc, err := s.resolveExecLocation(skill, sourceDir, isExistingDir(sourceDir), req.Isolate, req.DryRun, pin)
	if err != nil {
		return nil, err
	}
	if record != nil {
		record.WorkDir = loc.WorkDir
		record.Mode = loc.Mode
		record.SourceHash = s.sourceHashForAudit(loc, sourceDir)
	}

	manifest, err := LoadManifest(loc.ManifestDir)
	if err != nil {
		if errors.Is(err, ErrManifestNotFound) {
			return nil, ErrExecManifestMissing
		}
		return nil, err
	}
	if manifest.SetupCommand == "" {
		return nil, ErrExecSetupMissing
	}
	setupLine, declared := manifest.Scripts[manifest.SetupCommand]
	if !declared {
		return nil, fmt.Errorf("runtime.setup references unknown command %q", manifest.SetupCommand)
	}

	plan := &ExecPlan{
		WorkDir:      loc.WorkDir,
		Mode:         loc.Mode,
		CacheReused:  loc.CacheReused,
		Materialized: loc.Materialized,
		Pin:          pin,
		CommandLine:  setupLine,
		Setup:        manifest.SetupCommand,
	}
	if loc.Mode == "cache" {
		plan.SourceDir = sourceDir
	}
	if manifest.Deps.Declared {
		plan.Deps, plan.DepsSkipped = s.depsPlan(manifest, s.cacheDirFor(skill.Zid), loc.WorkDir)
	}
	entry := readSetupEntry(s.cacheDirFor(skill.Zid), loc.WorkDir)
	plan.SetupSkipped = setupEntryFresh(entry, manifest.Hash()) && !req.Force

	result = &ExecResult{
		SkillZid:  skill.Zid,
		SkillName: skill.Name,
		Command:   manifest.SetupCommand,
		WorkDir:   loc.WorkDir,
		Plan:      plan,
	}
	if req.DryRun {
		result.DryRun = true
		result.OK = true
		return result, nil
	}

	depsInfo, depsErr := s.ensureDeps(ctx, skill, manifest, loc.WorkDir, req.Stdout, req.Stderr)
	if depsErr != nil {
		return nil, depsErr
	}
	if depsInfo != nil {
		result.Deps = depsInfo
		if depsInfo.ExitCode != 0 || depsInfo.TimedOut {
			result.Aborted = "deps-failed"
			result.ExitCode = depsInfo.ExitCode
			result.TimedOut = depsInfo.TimedOut
			result.DurationMs = depsInfo.DurationMs
			return result, nil
		}
	}

	info, err := s.ensureSetup(ctx, skill, manifest, loc.WorkDir, req.Stdout, req.Stderr, req.Force)
	if err != nil {
		return nil, err
	}
	result.Setup = info
	result.ExitCode = info.ExitCode
	result.TimedOut = info.TimedOut
	result.DurationMs = info.DurationMs
	result.OK = info.ExitCode == 0 && !info.TimedOut
	return result, nil
}

// ensureSetup runs skm.runtime.setup when needed and maintains its
// completion marker. The marker binds the setup to both the working
// directory and the manifest hash, so source and cache executions keep
// separate setup state and manifest edits trigger a re-run. force skips the
// marker check. Concurrent first runs serialize on the exclusive zid lock;
// the freshness check repeats inside the lock so only one of them runs.
func (s *ExecService) ensureSetup(ctx context.Context, skill *models.Skill, manifest *ParsedManifest, workDir string, stdout, stderr io.Writer, force bool) (*SetupInfo, error) {
	setupName := manifest.SetupCommand
	setupLine, declared := manifest.Scripts[setupName]
	if !declared {
		return nil, fmt.Errorf("runtime.setup references unknown command %q", setupName)
	}

	cacheDir := s.cacheDirFor(skill.Zid)
	manifestHash := manifest.Hash()
	if !force && setupEntryFresh(readSetupEntry(cacheDir, workDir), manifestHash) {
		return &SetupInfo{Command: setupName, Skipped: true}, nil
	}

	lock, err := s.acquireExecLock(skill.Zid, false)
	if err != nil {
		return nil, err
	}
	defer lock.release()
	if !force && setupEntryFresh(readSetupEntry(cacheDir, workDir), manifestHash) {
		return &SetupInfo{Command: setupName, Skipped: true}, nil
	}

	annotation := findCommandAnnotation(manifest.Commands, setupName)
	timeout := time.Duration(0)
	if annotation != nil && annotation.TimeoutSeconds > 0 {
		timeout = time.Duration(annotation.TimeoutSeconds) * time.Second
	}

	// Setup installs what the skill needs; it runs with the standard SKM_*
	// environment but is not gated by the skill's required runtime env.
	env := buildExecEnv(workDir, skill, manifest, setupName, nil, "", nil)
	run, err := runLifecycleCommand(ctx, workDir, setupLine, env, timeout, stdout, stderr)
	if err != nil {
		return nil, fmt.Errorf("start setup command %q: %w", setupName, err)
	}

	info := &SetupInfo{
		Command:    setupName,
		Ran:        true,
		ExitCode:   run.ExitCode,
		TimedOut:   run.TimedOut,
		DurationMs: run.DurationMs,
		Output:     run.Output,
	}
	if run.TimedOut || run.ExitCode != 0 {
		return info, nil
	}

	if err := writeSetupEntry(cacheDir, workDir, setupMarkerEntry{
		ManifestHash: manifestHash,
		SetupCommand: setupName,
		CompletedAt:  time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("record setup completion: %w", err)
	}
	return info, nil
}

// lifecycleResult is the outcome of a managed lifecycle step (dependency
// installation, runtime.setup).
type lifecycleResult struct {
	ExitCode   int
	TimedOut   bool
	DurationMs int64
	// Output holds the captured stdout+stderr when the caller passed no
	// writers to stream to (e.g. --json mode).
	Output string
}

// runLifecycleCommand runs one managed lifecycle command (dependency install
// or runtime.setup) in workDir with the given environment. Output is always
// captured; when the caller passes writers it is also streamed. A non-zero
// exit or timeout is reported in the result, not as an error; only failing
// to start the process is an error.
func runLifecycleCommand(ctx context.Context, workDir, line string, env []string, timeout time.Duration, stdout, stderr io.Writer) (*lifecycleResult, error) {
	runCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(runCtx, "sh", "-c", line)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	cmd.WaitDelay = 5 * time.Second

	var captured bytes.Buffer
	cmd.Stdout = &captured
	cmd.Stderr = &captured
	streamed := false
	if stdout != nil {
		cmd.Stdout = io.MultiWriter(stdout, &captured)
		streamed = true
	}
	if stderr != nil {
		cmd.Stderr = io.MultiWriter(stderr, &captured)
		streamed = true
	}

	result := &lifecycleResult{}
	started := time.Now()
	runErr := cmd.Run()
	result.DurationMs = time.Since(started).Milliseconds()
	if !streamed {
		result.Output = captured.String()
	}

	if runCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.ExitCode = 124
		return result, nil
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return nil, runErr
	}
	return result, nil
}

var shellSafePattern = regexp.MustCompile(`^[A-Za-z0-9_./:@%+=,-]+$`)

// shellQuote quotes one value for safe inclusion in an sh command line. It
// is used for dry-run display only; actual execution passes arguments as
// positional parameters to sh and never interpolates them into the line.
func shellQuote(value string) string {
	if value != "" && shellSafePattern.MatchString(value) {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// FormatCommandLine renders the resolved command line for dry-run display.
// Long arguments (typically argv-mode JSON input) are truncated.
func (p *ExecPlan) FormatCommandLine() string {
	parts := []string{p.CommandLine}
	for _, arg := range p.Args {
		if len(arg) > 120 {
			arg = arg[:120] + "…"
		}
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}
