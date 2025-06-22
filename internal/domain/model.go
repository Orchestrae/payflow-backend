package domain

import (
	"time"

	"gorm.io/gorm"
)

// Model contains common fields for all tables
// Use this as an embedded struct in all domain models
// to get ID, CreatedAt, UpdatedAt, DeletedAt
type Model struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
