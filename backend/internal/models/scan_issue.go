package models

type ScanIssue struct {
	BaseModel
	ScanJobID    uint           `gorm:"index;not null" json:"-"`
	ScanJob      ScanJob        `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	ProviderID   *uint          `gorm:"index" json:"-"`
	Provider     *Provider      `gorm:"constraint:OnDelete:SET NULL" json:"provider,omitempty"`
	SkillID      *uint          `gorm:"index" json:"-"`
	Skill        *Skill         `gorm:"constraint:OnDelete:SET NULL" json:"skill,omitempty"`
	RootPath     string         `gorm:"type:varchar(1024);index;not null" json:"rootPath"`
	RelativePath string         `gorm:"type:varchar(1024)" json:"relativePath,omitempty"`
	Code         string         `gorm:"type:varchar(64);index;not null" json:"code"`
	Severity     string         `gorm:"type:varchar(16);index;not null" json:"severity"`
	Message      string         `gorm:"type:text;not null" json:"message"`
	Details      map[string]any `gorm:"serializer:json" json:"details,omitempty"`
}
