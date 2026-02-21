// internal/domain/cadre.go
package domain

type Cadre struct {
	Model
	BusinessID uint   `gorm:"index" json:"business_id"`
	Name       string `gorm:"size:255" json:"name"`

	// Relationships
	EarningComponents []EarningComponent `gorm:"foreignKey:CadreID" json:"earning_components"`
	DeductionRules    []DeductionRule    `gorm:"foreignKey:CadreID" json:"deduction_rules,omitempty"`
	Employees         []Employee         `gorm:"foreignKey:CadreID" json:"employees,omitempty"`
}

type EarningComponent struct {
	Model
	CadreID uint   `gorm:"index" json:"cadre_id"`
	Name    string `gorm:"size:255" json:"name"`
	Amount  int64  `gorm:"" json:"amount"`
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
	Model
	BusinessID       uint              `gorm:"index" json:"business_id"`
	CadreID          uint              `gorm:"index" json:"cadre_id,omitempty"`
	Name             string            `gorm:"size:255" json:"name"`
	Type             DeductionRuleType `gorm:"type:varchar(20)" json:"type"`
	Value            float64           `gorm:"" json:"value"`
	CalculationBasis CalculationBasis  `gorm:"type:varchar(20)" json:"calculation_basis"`
}
