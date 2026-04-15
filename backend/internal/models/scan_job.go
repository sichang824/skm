package models

import "time"

type ScanJob struct {
	BaseModel
	ProviderID    *uint      `gorm:"index" json:"-"`
	Provider      *Provider  `gorm:"constraint:OnDelete:SET NULL" json:"provider,omitempty"`
	Scope         string     `gorm:"type:varchar(32);index;not null" json:"scope"`
	StartedAt     time.Time  `gorm:"index;not null" json:"startedAt"`
	FinishedAt    *time.Time `gorm:"index" json:"finishedAt,omitempty"`
	Status        string     `gorm:"type:varchar(32);index;not null" json:"status"`
	AddedCount    int        `gorm:"not null;default:0" json:"addedCount"`
	RemovedCount  int        `gorm:"not null;default:0" json:"removedCount"`
	ChangedCount  int        `gorm:"not null;default:0" json:"changedCount"`
	InvalidCount  int        `gorm:"not null;default:0" json:"invalidCount"`
	ConflictCount int        `gorm:"not null;default:0" json:"conflictCount"`
	LogLines      []string   `gorm:"serializer:json" json:"logLines"`
}
