package service

import (
	"backend-go/internal/models"
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Audit tests (design doc §11.3): every non-dry-run invocation lands in
// exec_records exactly once through the single deferred choke point.

func execRecords(t *testing.T, service *ExecService) []models.ExecRecord {
	t.Helper()
	records := make([]models.ExecRecord, 0)
	if err := service.db.Order("id ASC").Find(&records).Error; err != nil {
		t.Fatalf("query exec records: %v", err)
	}
	return records
}

func execOnce(t *testing.T, service *ExecService, skill *models.Skill, command string) *ExecResult {
	t.Helper()
	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: command})
	if err != nil {
		t.Fatalf("Exec(%s) returned error: %v", command, err)
	}
	return result
}

var hexHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestExecAuditRecordsCompletedRun(t *testing.T) {
	service, skill, skillRoot := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	before := time.Now().Add(-time.Second)
	result := execOnce(t, service, skill, "plain-args")
	if !result.OK {
		t.Fatalf("expected ok run, got %+v", result)
	}

	records := execRecords(t, service)
	if len(records) != 1 {
		t.Fatalf("expected exactly 1 audit record, got %d", len(records))
	}
	record := records[0]
	if record.Status != "completed" || record.ExitCode != 0 || record.TimedOut {
		t.Fatalf("unexpected outcome fields: %+v", record)
	}
	if record.SkillZid != skill.Zid || record.SkillName != "fixture-skill" || record.Command != "plain-args" {
		t.Fatalf("unexpected identity fields: %+v", record)
	}
	if record.Trigger != "cli" || record.Who == "" || record.Mode != "source" || record.WorkDir != skillRoot {
		t.Fatalf("unexpected context fields: %+v", record)
	}
	if !hexHashPattern.MatchString(record.SourceHash) {
		t.Fatalf("source hash must be a 64-char hex tree hash, got %q", record.SourceHash)
	}
	if record.StartedAt.Before(before) || record.DurationMs < 0 {
		t.Fatalf("unexpected timing fields: %+v", record)
	}
	if record.Zid == "" {
		t.Fatal("audit record must get its own zid")
	}
}

func TestExecAuditRecordsArgsAndEnvKeysOnly(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	_, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "plain-args",
		Args:     []string{"first", "second arg"},
		Env:      []string{"SECRET_KEY=hunter2"},
	})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}

	records := execRecords(t, service)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	record := records[0]
	if len(record.Args) != 2 || record.Args[0] != "first" || record.Args[1] != "second arg" {
		t.Fatalf("args not recorded verbatim: %+v", record.Args)
	}
	if len(record.EnvKeys) != 1 || record.EnvKeys[0] != "SECRET_KEY" {
		t.Fatalf("env keys must be recorded, values never: %+v", record.EnvKeys)
	}

	// The serialized record must not leak the env value anywhere.
	blob, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if strings.Contains(string(blob), "hunter2") {
		t.Fatalf("audit record leaks env values: %s", blob)
	}
}

func TestExecAuditRecordsFailedAndTimeoutRuns(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	if result := execOnce(t, service, skill, "fails"); result.ExitCode != 3 {
		t.Fatalf("expected exit 3, got %+v", result)
	}
	if result := execOnce(t, service, skill, "sleeps"); !result.TimedOut {
		t.Fatalf("expected timeout, got %+v", result)
	}

	records := execRecords(t, service)
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Status != "failed" || records[0].ExitCode != 3 {
		t.Fatalf("failed run recorded wrong: %+v", records[0])
	}
	if records[1].Status != "timeout" || records[1].ExitCode != 124 || !records[1].TimedOut {
		t.Fatalf("timeout run recorded wrong: %+v", records[1])
	}
}

func TestExecAuditRecordsRejections(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)
	ctx := context.Background()

	rejected := []struct {
		name       string
		request    *ExecRequest
		reasonPart string
	}{
		{"unknown command", &ExecRequest{SkillZid: skill.Zid, Command: "nope"}, "not found"},
		{"confirm gate", &ExecRequest{SkillZid: skill.Zid, Command: "confirm-me"}, "confirmation"},
		{"input without declaration", &ExecRequest{SkillZid: skill.Zid, Command: "plain-args", InputJSON: []byte(`{"id":1}`)}, "no structured input"},
		{"invalid pin", &ExecRequest{SkillZid: skill.Zid, Command: "hello", Pin: "zz"}, "invalid --pin"},
		{"missing skill", &ExecRequest{SkillZid: "SKILdoesnotexist", Command: "hello"}, "not found"},
	}
	for _, testCase := range rejected {
		if _, err := service.Exec(ctx, testCase.request); err == nil {
			t.Fatalf("%s: expected rejection error", testCase.name)
		}
	}

	records := execRecords(t, service)
	if len(records) != len(rejected) {
		t.Fatalf("expected %d rejected records, got %d", len(rejected), len(records))
	}
	for index, testCase := range rejected {
		record := records[index]
		if record.Status != "rejected" {
			t.Fatalf("%s: status %q, want rejected", testCase.name, record.Status)
		}
		if !strings.Contains(record.Reason, testCase.reasonPart) {
			t.Fatalf("%s: reason %q does not contain %q", testCase.name, record.Reason, testCase.reasonPart)
		}
	}
	if records[len(rejected)-1].SkillName != "" {
		t.Fatalf("missing-skill rejection must have empty skill name, got %q", records[len(rejected)-1].SkillName)
	}
}

func TestExecAuditConfirmGatePassesWithAssumeYes(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "confirm-me", AssumeYes: true})
	if err != nil || !result.OK {
		t.Fatalf("confirm-me with --yes failed: %+v / %v", result, err)
	}
	records := execRecords(t, service)
	if len(records) != 1 || records[0].Status != "completed" {
		t.Fatalf("expected one completed record, got %+v", records)
	}
}

const auditSetupManifest = `{
	"name": "fixture-skill",
	"version": "1.0.0",
	"scripts": {
		"hello": "bash scripts/hello.sh",
		"setup": "echo setting-up"
	},
	"skm": { "schemaVersion": 1, "runtime": { "setup": "setup" } }
}`

const auditFailingSetupManifest = `{
	"name": "fixture-skill",
	"version": "1.0.0",
	"scripts": {
		"hello": "bash scripts/hello.sh",
		"setup": "exit 5"
	},
	"skm": { "schemaVersion": 1, "runtime": { "setup": "setup" } }
}`

func TestExecAuditRecordsSetupFailure(t *testing.T) {
	service, skill, _ := newExecFixture(t, auditFailingSetupManifest, "hello.sh", helloScript)
	result, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello"})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result.Aborted != "setup-failed" || result.ExitCode != 5 {
		t.Fatalf("expected setup-failed abort with exit 5, got %+v", result)
	}

	records := execRecords(t, service)
	if len(records) != 1 || records[0].Status != "setup_failed" || records[0].ExitCode != 5 {
		t.Fatalf("expected one setup_failed record, got %+v", records)
	}
}

func TestExecAuditRecordsDepsFailure(t *testing.T) {
	fixture := newDepsFixture(t, depsAutoManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
		"package-lock.json":  "{}",
	}, 9)

	result, err := fixture.service.Exec(context.Background(), &ExecRequest{SkillZid: fixture.skill.zid, Command: "toucher"})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result.Aborted != "deps-failed" {
		t.Fatalf("expected deps-failed abort, got %+v", result)
	}

	records := execRecords(t, fixture.service)
	if len(records) != 1 || records[0].Status != "deps_failed" || records[0].ExitCode != 9 {
		t.Fatalf("expected one deps_failed record with exit 9, got %+v", records)
	}
}

func TestExecAuditRunSetupUsesSentinelCommand(t *testing.T) {
	service, skill, _ := newExecFixture(t, auditSetupManifest, "hello.sh", helloScript)
	result, err := service.RunSetup(context.Background(), &SetupRequest{SkillZid: skill.Zid})
	if err != nil || !result.OK {
		t.Fatalf("RunSetup failed: %+v / %v", result, err)
	}

	records := execRecords(t, service)
	if len(records) != 1 || records[0].Command != "--setup" || records[0].Status != "completed" {
		t.Fatalf("expected one completed --setup record, got %+v", records)
	}
}

func TestExecAuditSkipsDryRuns(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	if _, err := service.Exec(context.Background(), &ExecRequest{SkillZid: skill.Zid, Command: "hello", DryRun: true}); err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if _, err := service.RunSetup(context.Background(), &SetupRequest{SkillZid: skill.Zid, DryRun: true}); err == nil {
		// This fixture declares no setup; the dry-run rejection is fine too —
		// either way nothing may be recorded.
	}
	if records := execRecords(t, service); len(records) != 0 {
		t.Fatalf("dry runs must not be audited, got %+v", records)
	}
}

func TestExecAuditRecordsTriggerAndPin(t *testing.T) {
	service, skill, skillRoot := newCacheFixture(t, isolateManifest, map[string]string{
		"scripts/toucher.sh": toucherScript,
	})
	sourceHash, err := hashSkillTree(skillRoot, materializationRules(skillRoot))
	if err != nil {
		t.Fatalf("hash source tree: %v", err)
	}

	result, err := service.Exec(context.Background(), &ExecRequest{
		SkillZid: skill.Zid,
		Command:  "toucher",
		Pin:      strings.ToUpper(sourceHash[:12]), // normalized to lowercase
		Trigger:  "http",
	})
	if err != nil || !result.OK {
		t.Fatalf("pinned http exec failed: %+v / %v", result, err)
	}

	records := execRecords(t, service)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	record := records[0]
	if record.Trigger != "http" || record.Pin != sourceHash[:12] {
		t.Fatalf("trigger/pin recorded wrong: %+v", record)
	}
	if record.SourceHash == "" || record.Mode != "cache" {
		t.Fatalf("pinned run must record cache mode and hash: %+v", record)
	}
}

func TestListExecRecordsFilterAndLimit(t *testing.T) {
	service, skill, _ := newExecFixture(t, execFixtureManifest, "hello.sh", helloScript)

	execOnce(t, service, skill, "hello")
	execOnce(t, service, skill, "hello")
	execOnce(t, service, skill, "plain-args")

	all, err := service.ListExecRecords(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("ListExecRecords: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 records, got %d", len(all))
	}
	for index := 1; index < len(all); index++ {
		if all[index-1].StartedAt.Before(all[index].StartedAt) {
			t.Fatal("records must be newest first")
		}
	}

	filtered, err := service.ListExecRecords(context.Background(), skill.Zid, 2)
	if err != nil {
		t.Fatalf("ListExecRecords filtered: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("limit not applied, got %d records", len(filtered))
	}
	for _, record := range filtered {
		if record.SkillZid != skill.Zid {
			t.Fatalf("filter leaked record of skill %s", record.SkillZid)
		}
	}

	none, err := service.ListExecRecords(context.Background(), "SKILdoesnotexist", 0)
	if err != nil {
		t.Fatalf("ListExecRecords missing skill: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no records for unknown skill, got %d", len(none))
	}
}
