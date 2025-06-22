// internal/domain/cadre.go
package domain

import (
	"time"

	"gorm.io/gorm"
)

type Cadre struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	BusinessID uint      `gorm:"index"`
	Name       string    `gorm:"size:255"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`

	// Relationships
	EarningComponents []EarningComponent `gorm:"foreignKey:CadreID"`
	DeductionRules    []DeductionRule    `gorm:"foreignKey:CadreID"`
	Employees         []Employee         `gorm:"foreignKey:CadreID"`
}

type EarningComponent struct {
	ID      uint   `gorm:"primaryKey;autoIncrement"`
	CadreID uint   `gorm:"index"`
	Name    string `gorm:"size:255"`
	Amount  int64  `gorm:""` // Use int64 for monetary values to avoid float inaccuracies
}

type DeductionRuleType string

const (
	DeductionTypePercentage DeductionRuleType = "percentage"
	DeductionTypeFlat       DeductionRuleType = "flat"
)

type CalculationBasis string

const (
	BasisGrossPay CalculationBasis = "gross_pay"
	BasisBasicPay CalculationBasis = "basic_pay" // Future-proofing
)

type DeductionRule struct {
	ID               uint              `gorm:"primaryKey;autoIncrement"`
	BusinessID       uint              `gorm:"index"`
	CadreID          uint              `gorm:"index"` // Optional: can be linked to specific cadre or global
	Name             string            `gorm:"size:255"`
	Type             DeductionRuleType `gorm:"type:varchar(20)"`
	Value            float64           `gorm:""` // Percentage (e.g., 7.5) or Flat Amount in smallest currency unit
	CalculationBasis CalculationBasis  `gorm:"type:varchar(20)"`
	CreatedAt        time.Time         `gorm:"autoCreateTime"`
	UpdatedAt        time.Time         `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt    `gorm:"index"`
}
