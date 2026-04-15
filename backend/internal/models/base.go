package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel defines common columns for all tables
type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"-"`
	Zid       string         `gorm:"type:varchar(32);uniqueIndex;not null" json:"zid"`
	CreatedAt time.Time      `gorm:"index" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"index" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

// BeforeCreate hook to prepare for zid generation
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.Zid != "" {
		return nil // Zid already set
	}
	return nil
}

// AfterCreate hook to generate zid after ID is assigned
func (b *BaseModel) AfterCreate(tx *gorm.DB) error {
	if b.Zid != "" {
		return nil // Zid already set
	}

	prefix := getModelPrefix(tx.Statement.Table)
	if prefix == "" {
		return nil // Skip if no prefix mapping
	}

	zid, err := Encode(prefix, uint64(b.ID))
	if err != nil {
		return err
	}

	b.Zid = zid
	return tx.Model(tx.Statement.Model).Update("zid", zid).Error
}

// getModelPrefix returns the zid prefix for a given table name
func getModelPrefix(tableName string) string {
	return GetPrefixForTable(tableName)
}
