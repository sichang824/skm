package service

import (
	"backend-go/internal/models"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ManifestFileName is the executable manifest file of a skill. It reuses the
// package.json convention: `scripts` is the command registry (single source
// of truth), and the optional `skm` section annotates commands with
// confirmation, timeout, env, and structured-input metadata.
const ManifestFileName = "package.json"

// ErrManifestNotFound is returned when a skill directory has no package.json.
var ErrManifestNotFound = errors.New("manifest not found")

// ParsedManifest is the parsed executable manifest of one skill directory.
type ParsedManifest struct {
	Name          string
	Version       string
	SchemaVersion int
	RuntimeEnv    []string
	SetupCommand  string
	// Deps is the managed dependency installation declaration
	// (skm.runtime.deps); Declared is the opt-in gate.
	Deps ManifestDeps
	// HasNPMDependencies reports whether package.json declares dependencies
	// or devDependencies (used to decide npm install when no lockfile
	// exists).
	HasNPMDependencies bool
	// Scripts maps command name -> command line (the package.json scripts
	// section). Only commands declared here are executable via skm.
	Scripts map[string]string
	// Commands are the scripts entries annotated with skm metadata, sorted
	// by name for stable display.
	Commands []models.SkillCommand
	// InputSchemas maps command name -> raw JSON Schema, kept for exec-time
	// input validation (the DB snapshot only records presence).
	InputSchemas map[string]json.RawMessage
	// leftoverAnnotations lists skm.commands names without a matching
	// scripts entry; only meaningful during scan validation.
	leftoverAnnotations []string
	// raw is the exact package.json content, kept for Hash.
	raw []byte
}

// ManifestDeps is the skm.runtime.deps declaration: skm-managed dependency
// installation is opt-in — only skills that declare the key get installs.
// An empty object ({}) auto-detects both supported ecosystems; explicit
// true values select ecosystems individually.
type ManifestDeps struct {
	Declared   bool     `json:"declared,omitempty"`
	Node       bool     `json:"node,omitempty"`
	Python     bool     `json:"python,omitempty"`
	AutoDetect bool     `json:"autoDetect,omitempty"`
	Unknown    []string `json:"unknown,omitempty"` // unknown keys / non-bool values (sorted)
}

// Resolve decides which ecosystems to manage given the files actually
// present in the working directory.
func (d ManifestDeps) Resolve(lockfileExists, npmDeps, requirementsExists bool) (node, python bool) {
	if !d.Declared {
		return false, false
	}
	node = d.Node || (d.AutoDetect && (lockfileExists || npmDeps))
	python = d.Python || (d.AutoDetect && requirementsExists)
	return node, python
}

// parseManifestDeps normalizes the raw skm.runtime.deps object. Recognized
// keys with bool values select ecosystems; everything else is collected as
// Unknown so scan can report it (a typo'd key silently disables managed
// installs otherwise).
func parseManifestDeps(raw map[string]any) ManifestDeps {
	deps := ManifestDeps{Declared: true}
	unknown := make([]string, 0)
	names := make([]string, 0, len(raw))
	for name := range raw {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		value, isBool := raw[name].(bool)
		switch name {
		case "node":
			if isBool {
				deps.Node = value
			} else {
				unknown = append(unknown, name)
			}
		case "python":
			if isBool {
				deps.Python = value
			} else {
				unknown = append(unknown, name)
			}
		default:
			unknown = append(unknown, name)
		}
	}
	deps.Unknown = unknown
	deps.AutoDetect = !deps.Node && !deps.Python
	return deps
}

// Hash identifies the manifest content (sha256 of the raw package.json).
// runtime.setup idempotency binds to it: setup re-runs only when the
// manifest changes.
func (m *ParsedManifest) Hash() string {
	sum := sha256.Sum256(m.raw)
	return hex.EncodeToString(sum[:])
}

type manifestFile struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Skm             *manifestSkm      `json:"skm"`
}

type manifestSkm struct {
	SchemaVersion int                          `json:"schemaVersion"`
	Runtime       *manifestRuntime             `json:"runtime"`
	Commands      map[string]manifestCommandAn `json:"commands"`
}

type manifestRuntime struct {
	Env   []string       `json:"env"`
	Setup string         `json:"setup"`
	Deps  map[string]any `json:"deps"`
}

// manifestCommandAn is the annotation of one command in the skm section.
// Confirm accepts both a bool and a string in JSON.
type manifestCommandAn struct {
	Description    string         `json:"description"`
	Confirm        any            `json:"confirm"`
	TimeoutSeconds int            `json:"timeoutSeconds"`
	Env            []string       `json:"env"`
	Input          *manifestInput `json:"input"`
}

type manifestInput struct {
	Via    string          `json:"via"`
	Schema json.RawMessage `json:"schema"`
}

// ManifestPath returns the absolute path of the manifest inside a skill
// directory.
func ManifestPath(skillRoot string) string {
	return filepath.Join(skillRoot, ManifestFileName)
}

// LoadManifest reads and parses the package.json of a skill directory. It
// returns ErrManifestNotFound when the file does not exist, and a parse error
// (reported as the manifest_invalid_json scan issue) when it is invalid.
func LoadManifest(skillRoot string) (*ParsedManifest, error) {
	data, err := os.ReadFile(ManifestPath(skillRoot))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrManifestNotFound
		}
		return nil, fmt.Errorf("read %s: %w", ManifestFileName, err)
	}
	return ParseManifest(data)
}

// ParseManifest parses package.json content into a ParsedManifest.
func ParseManifest(data []byte) (*ParsedManifest, error) {
	var file manifestFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ManifestFileName, err)
	}

	manifest := &ParsedManifest{
		Name:               strings.TrimSpace(file.Name),
		Version:            strings.TrimSpace(file.Version),
		HasNPMDependencies: len(file.Dependencies) > 0 || len(file.DevDependencies) > 0,
		Scripts:            map[string]string{},
		InputSchemas:       map[string]json.RawMessage{},
		raw:                data,
	}
	for name, line := range file.Scripts {
		name = strings.TrimSpace(name)
		line = strings.TrimSpace(line)
		if name == "" || line == "" {
			continue
		}
		manifest.Scripts[name] = line
	}

	annotations := map[string]manifestCommandAn{}
	if file.Skm != nil {
		manifest.SchemaVersion = file.Skm.SchemaVersion
		if file.Skm.Runtime != nil {
			manifest.RuntimeEnv = normalizeEnvNames(file.Skm.Runtime.Env)
			manifest.SetupCommand = strings.TrimSpace(file.Skm.Runtime.Setup)
			if file.Skm.Runtime.Deps != nil {
				manifest.Deps = parseManifestDeps(file.Skm.Runtime.Deps)
			}
		}
		for name, annotation := range file.Skm.Commands {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			annotations[name] = annotation
		}
	}

	names := make([]string, 0, len(manifest.Scripts))
	for name := range manifest.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		command := models.SkillCommand{
			Name: name,
			Line: manifest.Scripts[name],
		}
		if annotation, ok := annotations[name]; ok {
			command.Description = strings.TrimSpace(annotation.Description)
			command.Confirm = normalizeConfirm(annotation.Confirm)
			command.TimeoutSeconds = annotation.TimeoutSeconds
			command.Env = normalizeEnvNames(annotation.Env)
			if annotation.Input != nil {
				command.InputVia = normalizeInputVia(annotation.Input.Via)
				schema := annotation.Input.Schema
				if len(schema) > 0 && string(schema) != "null" {
					command.HasInputSchema = true
					manifest.InputSchemas[name] = schema
				}
			}
			delete(annotations, name)
		}
		manifest.Commands = append(manifest.Commands, command)
	}

	// Leftover annotations reference commands that do not exist in scripts;
	// keep them sorted so validation reports them deterministically.
	manifest.leftoverAnnotations = make([]string, 0, len(annotations))
	for name := range annotations {
		manifest.leftoverAnnotations = append(manifest.leftoverAnnotations, name)
	}
	sort.Strings(manifest.leftoverAnnotations)

	return manifest, nil
}

// Validate cross-checks the manifest against the skill directory and the
// SKILL.md identity, returning scan issues and issue codes for the skill
// record. Manifest issues never flip the skill status to invalid: SKILL.md
// stays usable, and the issues list carries the severity.
func (m *ParsedManifest) Validate(skillRoot string, skillName string) ([]discoveredIssue, []string) {
	issues := make([]discoveredIssue, 0)
	codes := make([]string, 0)
	add := func(code, severity, message string, details map[string]any) {
		issues = append(issues, discoveredIssue{
			RootPath:     skillRoot,
			RelativePath: ManifestFileName,
			Code:         code,
			Severity:     severity,
			Message:      message,
			Details:      details,
			SkillRoot:    skillRoot,
		})
		if !containsString(codes, code) {
			codes = append(codes, code)
		}
	}

	for _, name := range m.leftoverAnnotations {
		add("manifest_command_missing", "error",
			fmt.Sprintf("skm.commands entry %q has no matching scripts entry", name),
			map[string]any{"command": name})
	}
	if m.SetupCommand != "" {
		if _, ok := m.Scripts[m.SetupCommand]; !ok {
			add("manifest_command_missing", "error",
				fmt.Sprintf("skm.runtime.setup references unknown command %q", m.SetupCommand),
				map[string]any{"command": m.SetupCommand})
		}
	}

	for _, key := range m.Deps.Unknown {
		add("manifest_deps_unknown", "error",
			fmt.Sprintf("skm.runtime.deps entry %q is not a recognized ecosystem (node, python); a typo silently disables managed installs", key),
			map[string]any{"key": key})
	}

	for _, command := range m.Commands {
		for _, target := range manifestTargetRefs(command.Line) {
			if _, err := os.Stat(filepath.Join(skillRoot, filepath.FromSlash(target))); err != nil {
				add("manifest_target_missing", "warning",
					fmt.Sprintf("command %q references missing file %s", command.Name, target),
					map[string]any{"command": command.Name, "target": target})
			}
		}
	}

	if m.Name != "" && skillName != "" && slugify(m.Name) != slugify(skillName) {
		add("manifest_name_mismatch", "warning",
			fmt.Sprintf("package.json name %q does not match SKILL.md name %q", m.Name, skillName),
			map[string]any{"manifestName": m.Name, "skillName": skillName})
	}

	return issues, codes
}

// manifestTargetRefs extracts relative file references (scripts/... paths)
// from a command line. This is a lightweight heuristic: tokens containing a
// scripts/ path segment are checked for existence at scan time.
func manifestTargetRefs(line string) []string {
	refs := make([]string, 0)
	seen := map[string]struct{}{}
	for _, token := range strings.Fields(line) {
		token = strings.Trim(token, "\"'`,;()[]{}")
		token = strings.TrimPrefix(token, "./")
		if !strings.HasPrefix(token, "scripts/") {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		refs = append(refs, token)
	}
	return refs
}

func normalizeConfirm(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case bool:
		if typed {
			return "requires confirmation before running"
		}
		return ""
	case string:
		return strings.TrimSpace(typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func normalizeEnvNames(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func normalizeInputVia(via string) string {
	switch strings.ToLower(strings.TrimSpace(via)) {
	case "", "stdin":
		return "stdin"
	case "argv", "args":
		return "argv"
	case "env":
		return "env"
	default:
		return strings.ToLower(strings.TrimSpace(via))
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
