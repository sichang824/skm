package service

import (
	"backend-go/internal/models"
	"context"
	"os"
	"os/user"
	"strings"
	"time"
)

// Exec audit (design doc §11.3): every non-dry-run invocation lands in the
// exec_records table through a single deferred choke point, covering every
// return path including pre-check rejections. Recording is best effort: a
// database failure never breaks an exec.

// setupSentinelCommand is the ExecRecord.Command value of RunSetup records.
const setupSentinelCommand = "--setup"

// newExecRecord starts an audit record, or returns nil when the invocation
// is not audited (dry runs).
func (s *ExecService) newExecRecord(dryRun bool, skillZid, command, trigger string, args, env []string, pin string) *models.ExecRecord {
	if dryRun {
		return nil
	}
	if trigger == "" {
		trigger = "cli"
	}
	return &models.ExecRecord{
		SkillZid:  skillZid,
		Command:   command,
		Trigger:   trigger,
		Who:       currentUsername(),
		Pin:       pin,
		Args:      append([]string{}, args...),
		EnvKeys:   envKeyNames(env),
		StartedAt: time.Now(),
	}
}

// finalizeExecRecord derives the terminal status from the outcome and
// persists the record. Dry runs never reach this point (newExecRecord is
// skipped for them by the callers).
func (s *ExecService) finalizeExecRecord(ctx context.Context, record *models.ExecRecord, result *ExecResult, retErr error) {
	if s.db == nil || record == nil {
		return
	}
	switch {
	case retErr != nil:
		record.Status = "rejected"
		record.Reason = truncateReason(retErr.Error())
	case result.Aborted == "setup-failed":
		record.Status = "setup_failed"
		record.ExitCode = result.ExitCode
		record.TimedOut = result.TimedOut
		record.DurationMs = result.DurationMs
		record.Reason = truncateReason("runtime.setup exited non-zero; command not started")
	case result.Aborted == "deps-failed":
		record.Status = "deps_failed"
		record.ExitCode = result.ExitCode
		record.TimedOut = result.TimedOut
		record.DurationMs = result.DurationMs
		record.Reason = truncateReason("managed dependency installation failed; command not started")
	case result.TimedOut:
		record.Status = "timeout"
		record.ExitCode = result.ExitCode
		record.TimedOut = true
		record.DurationMs = result.DurationMs
	case result.ExitCode == 0:
		record.Status = "completed"
		record.DurationMs = result.DurationMs
	default:
		record.Status = "failed"
		record.ExitCode = result.ExitCode
		record.DurationMs = result.DurationMs
	}
	// Best effort: auditing must never fail the exec itself.
	_ = s.db.WithContext(ctx).Create(record).Error
}

// sourceHashForAudit returns the tree hash that identifies the content a
// run used: the cache copy's recorded hash, or a best-effort hash of the
// source tree in source mode (empty when it cannot be computed — the run is
// unaffected).
func (s *ExecService) sourceHashForAudit(loc *execLocation, sourceDir string) string {
	if loc.SourceHash != "" {
		return loc.SourceHash
	}
	if loc.Mode == "source" {
		if hash, err := hashSkillTree(sourceDir, materializationRules(sourceDir)); err == nil {
			return hash
		}
	}
	return ""
}

// currentUsername resolves the OS user on a best-effort basis.
func currentUsername() string {
	if current, err := user.Current(); err == nil && strings.TrimSpace(current.Username) != "" {
		return current.Username
	}
	return os.Getenv("USER")
}

// envKeyNames extracts variable names from KEY=VAL entries. Values are
// never recorded: secrets must not enter the audit trail.
func envKeyNames(entries []string) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if name, _, found := strings.Cut(entry, "="); found && name != "" {
			names = append(names, name)
		}
	}
	return names
}

func truncateReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) > 250 {
		return reason[:250] + "…"
	}
	return reason
}

// ListExecRecords returns audit records newest-first, optionally filtered to
// one skill. limit defaults to 20 and is capped at 200.
func (s *ExecService) ListExecRecords(ctx context.Context, skillZid string, limit int) ([]models.ExecRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	query := s.db.WithContext(ctx).Model(&models.ExecRecord{}).Order("started_at DESC, id DESC").Limit(limit)
	if zid := strings.TrimSpace(skillZid); zid != "" {
		query = query.Where("skill_zid = ?", zid)
	}
	records := make([]models.ExecRecord, 0, limit)
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}
