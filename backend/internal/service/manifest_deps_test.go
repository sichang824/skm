package service

import (
	"testing"
)

// Dependency-declaration parsing (skm.runtime.deps) is opt-in: the key's
// presence is the gate, explicit true values select ecosystems, an empty
// object auto-detects, and unknown keys surface as scan issues.

func parseManifestForDeps(t *testing.T, body string) *ParsedManifest {
	t.Helper()
	manifest, err := ParseManifest([]byte(body))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	return manifest
}

func TestParseManifestDepsNotDeclared(t *testing.T) {
	manifest := parseManifestForDeps(t, `{"scripts": {"a": "echo hi"}}`)
	if manifest.Deps.Declared {
		t.Fatalf("deps must be undeclared when the key is absent: %+v", manifest.Deps)
	}
	if node, python := manifest.Deps.Resolve(true, true, true); node || python {
		t.Fatalf("undeclared deps must manage nothing, got node=%v python=%v", node, python)
	}
}

func TestParseManifestDepsExplicitEcosystems(t *testing.T) {
	manifest := parseManifestForDeps(t, `{
		"scripts": {"a": "echo hi"},
		"skm": {"runtime": {"deps": {"node": true}}}
	}`)
	if !manifest.Deps.Declared || !manifest.Deps.Node || manifest.Deps.Python || manifest.Deps.AutoDetect {
		t.Fatalf("unexpected deps parse: %+v", manifest.Deps)
	}

	node, python := manifest.Deps.Resolve(false, false, true)
	if !node || python {
		t.Fatalf("explicit node:true must manage node regardless of files, got node=%v python=%v", node, python)
	}
}

func TestParseManifestDepsAutoDetect(t *testing.T) {
	manifest := parseManifestForDeps(t, `{
		"scripts": {"a": "echo hi"},
		"skm": {"runtime": {"deps": {}}}
	}`)
	if !manifest.Deps.Declared || !manifest.Deps.AutoDetect || manifest.Deps.Node || manifest.Deps.Python {
		t.Fatalf("unexpected deps parse: %+v", manifest.Deps)
	}

	cases := []struct {
		name                string
		lockfile, npm, reqs bool
		wantNode, wantPy    bool
	}{
		{"nothing present", false, false, false, false, false},
		{"lockfile only", true, false, false, true, false},
		{"npm deps only", false, true, false, true, false},
		{"requirements only", false, false, true, false, true},
		{"everything", true, true, true, true, true},
	}
	for _, testCase := range cases {
		node, python := manifest.Deps.Resolve(testCase.lockfile, testCase.npm, testCase.reqs)
		if node != testCase.wantNode || python != testCase.wantPy {
			t.Fatalf("%s: got node=%v python=%v, want node=%v python=%v",
				testCase.name, node, python, testCase.wantNode, testCase.wantPy)
		}
	}
}

func TestParseManifestDepsUnknownAndNonBool(t *testing.T) {
	manifest := parseManifestForDeps(t, `{
		"scripts": {"a": "echo hi"},
		"skm": {"runtime": {"deps": {"nodes": true, "python": "yes", "node": true}}}
	}`)
	if !manifest.Deps.Node {
		t.Fatalf("valid node key must still parse: %+v", manifest.Deps)
	}
	if len(manifest.Deps.Unknown) != 2 || manifest.Deps.Unknown[0] != "nodes" || manifest.Deps.Unknown[1] != "python" {
		t.Fatalf("unknown keys must be collected sorted: %+v", manifest.Deps.Unknown)
	}
}

func TestParseManifestHasNPMDependencies(t *testing.T) {
	withDeps := parseManifestForDeps(t, `{"dependencies": {"left-pad": "1.0.0"}, "scripts": {"a": "echo"}}`)
	if !withDeps.HasNPMDependencies {
		t.Fatal("dependencies section must set HasNPMDependencies")
	}
	withDev := parseManifestForDeps(t, `{"devDependencies": {"vitest": "3.0.0"}, "scripts": {"a": "echo"}}`)
	if !withDev.HasNPMDependencies {
		t.Fatal("devDependencies section must set HasNPMDependencies")
	}
	without := parseManifestForDeps(t, `{"scripts": {"a": "echo"}}`)
	if without.HasNPMDependencies {
		t.Fatal("no dependency sections must leave HasNPMDependencies false")
	}
}

func TestManifestValidateDepsUnknownIssue(t *testing.T) {
	manifest := parseManifestForDeps(t, `{
		"name": "fixture-skill",
		"scripts": {"a": "echo hi"},
		"skm": {"runtime": {"deps": {"nodee": true}}}
	}`)
	issues, codes := manifest.Validate(t.TempDir(), "fixture-skill")

	found := false
	for _, issue := range issues {
		if issue.Code == "manifest_deps_unknown" {
			found = true
			if issue.Severity != "error" {
				t.Fatalf("manifest_deps_unknown must be an error, got %q", issue.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("expected manifest_deps_unknown issue, got %+v", issues)
	}
	if !containsString(codes, "manifest_deps_unknown") {
		t.Fatalf("issue codes must include manifest_deps_unknown, got %v", codes)
	}
}
