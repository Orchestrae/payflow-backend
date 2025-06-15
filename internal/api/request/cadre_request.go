package request

import "payflow/internal/domain"

type CreateCadreRequest struct {
	Name              string                    `json:"name" validate:"required"`
	EarningComponents []domain.EarningComponent `json:"earning_components" validate:"required"`
	DeductionRuleIDs  []uint                    `json:"deduction_rule_ids"`
}

type UpdateCadreRequest struct {
	Name              string                    `json:"name" validate:"required"`
	EarningComponents []domain.EarningComponent `json:"earning_components" validate:"required"`
	DeductionRuleIDs  []uint                    `json:"deduction_rule_ids"`
}
