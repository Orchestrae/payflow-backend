// internal/domain/cadre.go
package domain

import (
	"gorm.io/gorm"
	"time"
)

type Cadre struct {
	ID         uint
	BusinessID uint
	Name       string
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// Relational fields
	EarningComponents []EarningComponent
	DeductionRules    []DeductionRule
}

type EarningComponent struct {
	ID      uint
	CadreID uint
	Name    string
	Amount  int64 // Use int64 for monetary values to avoid float inaccuracies
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
	gorm.Model
	BusinessID       uint
	Name             string
	Type             DeductionRuleType
	Value            float64 // Percentage (e.g., 7.5) or Flat Amount in smallest currency unit
	CalculationBasis CalculationBasis
}
