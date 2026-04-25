package model

import (
	"time"

	"gorm.io/gorm"
)

type RefreshToken struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UserID    string         `gorm:"size:128;index;not null" json:"user_id"`
	TokenID   string         `gorm:"size:512;uniqueIndex;not null" json:"token_id"`
	Device    string         `gorm:"size:64" json:"device"`
	UserAgent string         `gorm:"size:512" json:"user_agent"`
	ExpiresAt time.Time      `gorm:"index" json:"expires_at"`
	Revoked   bool           `gorm:"default:false" json:"revoked"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

func AutoMigrateRefreshToken(db *gorm.DB) error {
	return db.AutoMigrate(&RefreshToken{})
}
