package domain

import (
	"time"

	"gorm.io/gorm"
)

// Model contains common fields for all tables
// Use this as an embedded struct in all domain models
// to get ID, CreatedAt, UpdatedAt, DeletedAt
type Model struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
