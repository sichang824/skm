package models

import "time"

// ExecRecord is one audited exec invocation (design doc §11.3). Records are
// intentionally FK-free and denormalized: they survive skill/provider
// deletion so the audit trail remains the history of what actually ran.
// Secrets are never stored: EnvKeys holds variable names only and structured
// input is not recorded; Args are recorded because command arguments are
// part of "what ran" (skills must not take secrets via argv).
type ExecRecord struct {
	BaseModel
	SkillZid  string `gorm:"type:varchar(32);index;not null" json:"skillZid"`
	SkillName string `gorm:"type:varchar(255);index" json:"skillName"`
	// Command is the declared command name; RunSetup invocations use the
	// "--setup" sentinel.
	Command string `gorm:"type:varchar(255);index;not null" json:"command"`
	// Trigger records where the invocation came from: "cli" or "http".
	Trigger string `gorm:"type:varchar(16);index;not null" json:"trigger"`
	// Who is the OS user that ran the command (best effort).
	Who     string `gorm:"type:varchar(255)" json:"who,omitempty"`
	WorkDir string `gorm:"type:varchar(1024)" json:"workDir,omitempty"`
	// Mode is "source" or "cache".
	Mode string `gorm:"type:varchar(16)" json:"mode,omitempty"`
	// Pin is the requested version pin (normalized lowercase hex), if any.
	Pin string `gorm:"type:varchar(64);index" json:"pin,omitempty"`
	// SourceHash is the tree hash of the content that ran; it is the value
	// a future --pin can replay.
	SourceHash string `gorm:"type:varchar(64);index" json:"sourceHash,omitempty"`
	// Status is one of completed, failed, timeout, setup_failed,
	// deps_failed, rejected.
	Status   string `gorm:"type:varchar(32);index;not null" json:"status"`
	ExitCode int    `gorm:"not null;default:0" json:"exitCode"`
	TimedOut bool   `gorm:"not null;default:false" json:"timedOut"`
	// Reason explains rejections and aborts (truncated to fit the column).
	Reason     string   `gorm:"type:varchar(255)" json:"reason,omitempty"`
	Args       []string `gorm:"serializer:json" json:"args,omitempty"`
	EnvKeys    []string `gorm:"serializer:json" json:"envKeys,omitempty"`
	InputVia   string   `gorm:"type:varchar(16)" json:"inputVia,omitempty"`
	DurationMs int64    `gorm:"not null;default:0" json:"durationMs"`
	StartedAt  time.Time `gorm:"index;not null" json:"startedAt"`
}
