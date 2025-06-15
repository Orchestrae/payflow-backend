package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/repository"
)

// DeductionRuleService defines the interface for deduction rule-related business logic.
type DeductionRuleService interface {
	CreateDeductionRule(ctx context.Context, rule *domain.DeductionRule) (*domain.DeductionRule, error)
	ListByBusinessID(ctx context.Context, businessID uint) ([]domain.DeductionRule, error)
	GetByID(ctx context.Context, ruleID, businessID uint) (*domain.DeductionRule, error)
	UpdateDeductionRule(ctx context.Context, rule *domain.DeductionRule) (*domain.DeductionRule, error)
	DeleteDeductionRule(ctx context.Context, ruleID, businessID uint) error
}

// deductionRuleService is the concrete implementation of the DeductionRuleService.
type deductionRuleService struct {
	deductionRuleRepo repository.DeductionRuleRepository
}

// NewDeductionRuleService creates a new instance of the deduction rule service.
func NewDeductionRuleService(deductionRuleRepo repository.DeductionRuleRepository) DeductionRuleService {
	return &deductionRuleService{
		deductionRuleRepo: deductionRuleRepo,
	}
}

// CreateDeductionRule validates and creates a new deduction rule.
func (s *deductionRuleService) CreateDeductionRule(ctx context.Context, rule *domain.DeductionRule) (*domain.DeductionRule, error) {
	if err := s.deductionRuleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create deduction rule: %w", err)
	}
	return rule, nil
}

// ListByBusinessID retrieves all deduction rules for a business.
func (s *deductionRuleService) ListByBusinessID(ctx context.Context, businessID uint) ([]domain.DeductionRule, error) {
	return s.deductionRuleRepo.FindAllByBusinessID(ctx, businessID)
}

// GetByID retrieves a specific deduction rule by ID, ensuring it belongs to the specified business.
func (s *deductionRuleService) GetByID(ctx context.Context, ruleID, businessID uint) (*domain.DeductionRule, error) {
	rule, err := s.deductionRuleRepo.FindByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}

	if rule.BusinessID != businessID {
		return nil, domain.ErrForbidden
	}

	return rule, nil
}

// UpdateDeductionRule updates an existing deduction rule.
func (s *deductionRuleService) UpdateDeductionRule(ctx context.Context, rule *domain.DeductionRule) (*domain.DeductionRule, error) {
	if err := s.deductionRuleRepo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update deduction rule: %w", err)
	}
	return rule, nil
}

// DeleteDeductionRule deletes a deduction rule, ensuring it belongs to the specified business.
func (s *deductionRuleService) DeleteDeductionRule(ctx context.Context, ruleID, businessID uint) error {
	rule, err := s.deductionRuleRepo.FindByID(ctx, ruleID)
	if err != nil {
		return err
	}

	if rule.BusinessID != businessID {
		return domain.ErrForbidden
	}

	return s.deductionRuleRepo.Delete(ctx, ruleID)
}
