package models

import "time"

type SkillRelation struct {
	Mode        string   `json:"mode"`
	FromPath    string   `json:"fromPath,omitempty"`
	Directories []string `json:"directories,omitempty"`
	Include     []string `json:"include,omitempty"`
	Exclude     []string `json:"exclude,omitempty"`
}

// SkillCommand is one executable command declared in a skill's package.json
// (scripts entry, annotated by the skm section). It is a discovery snapshot:
// the exec engine re-reads package.json from disk at execution time.
type SkillCommand struct {
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	Line           string   `json:"line"`
	Confirm        string   `json:"confirm,omitempty"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"`
	Env            []string `json:"env,omitempty"`
	InputVia       string   `json:"inputVia,omitempty"`
	HasInputSchema bool     `json:"hasInputSchema,omitempty"`
}

type Skill struct {
	BaseModel
	ProviderID     uint           `gorm:"uniqueIndex:idx_skills_provider_root;index;not null" json:"-"`
	Provider       Provider       `gorm:"constraint:OnDelete:CASCADE" json:"provider,omitempty"`
	Name           string         `gorm:"type:varchar(255);index;not null" json:"name"`
	Slug           string         `gorm:"type:varchar(255);index;not null" json:"slug"`
	DirectoryName  string         `gorm:"type:varchar(255);index;not null" json:"directoryName"`
	RootPath       string         `gorm:"type:varchar(1024);index;not null;uniqueIndex:idx_skills_provider_root" json:"rootPath"`
	SkillMdPath    string         `gorm:"type:varchar(1024)" json:"skillMdPath,omitempty"`
	Category       string         `gorm:"type:varchar(255);index" json:"category,omitempty"`
	Tags           []string       `gorm:"serializer:json" json:"tags"`
	Summary        string         `gorm:"type:text" json:"summary,omitempty"`
	Status         string         `gorm:"type:varchar(32);index;not null" json:"status"`
	ContentHash    string         `gorm:"type:varchar(64);index" json:"contentHash,omitempty"`
	LastModifiedAt *time.Time     `gorm:"index" json:"lastModifiedAt,omitempty"`
	LastScannedAt  time.Time      `gorm:"index;not null" json:"lastScannedAt"`
	RawMarkdown    string         `gorm:"type:text" json:"rawMarkdown,omitempty"`
	BodyMarkdown   string         `gorm:"type:text" json:"bodyMarkdown,omitempty"`
	Frontmatter    map[string]any `gorm:"serializer:json" json:"frontmatter,omitempty"`
	Commands       []SkillCommand `gorm:"serializer:json" json:"commands,omitempty"`
	IssueCodes     []string       `gorm:"serializer:json" json:"issueCodes"`
	ConflictKinds  []string       `gorm:"serializer:json" json:"conflictKinds"`
	IsConflict     bool           `gorm:"index;not null;default:false" json:"isConflict"`
	IsEffective    bool           `gorm:"index;not null;default:true" json:"isEffective"`
	Relation       *SkillRelation `gorm:"-" json:"relation,omitempty"`
	RelatedSkills  []Skill        `gorm:"-" json:"relatedSkills,omitempty"`
}
