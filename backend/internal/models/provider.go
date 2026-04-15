package models

import "time"

type Provider struct {
	BaseModel
	Name            string     `gorm:"type:varchar(120);uniqueIndex;not null" json:"name"`
	Type            string     `gorm:"type:varchar(64);index;not null" json:"type"`
	RootPath        string     `gorm:"type:varchar(1024);uniqueIndex;not null" json:"rootPath"`
	Enabled         bool       `gorm:"index;not null;default:true" json:"enabled"`
	Priority        int        `gorm:"index;not null;default:100" json:"priority"`
	ScanMode        string     `gorm:"type:varchar(32);not null;default:'recursive'" json:"scanMode"`
	Description     string     `gorm:"type:text" json:"description,omitempty"`
	LastScannedAt   *time.Time `gorm:"index" json:"lastScannedAt,omitempty"`
	LastScanStatus  string     `gorm:"type:varchar(32);index;not null;default:'never'" json:"lastScanStatus"`
	LastScanSummary string     `gorm:"type:text" json:"lastScanSummary,omitempty"`
}
