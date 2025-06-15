package request

import "payflow/internal/domain"

type CreateDeductionRuleRequest struct {
	Name             string                   `json:"name" validate:"required"`
	Type             domain.DeductionRuleType `json:"type" validate:"required,oneof=percentage flat"`
	Value            float64                  `json:"value" validate:"required"`
	CalculationBasis domain.CalculationBasis  `json:"calculation_basis" validate:"required,oneof=gross_pay basic_pay"`
}

type UpdateDeductionRuleRequest struct {
	Name             string                   `json:"name" validate:"required"`
	Type             domain.DeductionRuleType `json:"type" validate:"required,oneof=percentage flat"`
	Value            float64                  `json:"value" validate:"required"`
	CalculationBasis domain.CalculationBasis  `json:"calculation_basis" validate:"required,oneof=gross_pay basic_pay"`
}
