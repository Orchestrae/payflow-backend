package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/platform/cache"
	"payflow/internal/repository"
	"time"
)

// DeductionRuleService defines the interface for deduction rule-related business logic.
type DeductionRuleService interface {
	CreateDeductionRule(ctx context.Context, rule *domain.DeductionRule) (*domain.DeductionRule, error)
	ListByBusinessID(ctx context.Context, businessID uint) ([]*domain.DeductionRule, error)
	GetByID(ctx context.Context, ruleID, businessID uint) (*domain.DeductionRule, error)
	UpdateDeductionRule(ctx context.Context, rule *domain.DeductionRule) (*domain.DeductionRule, error)
	DeleteDeductionRule(ctx context.Context, ruleID, businessID uint) error
}

// deductionRuleService is the concrete implementation of the DeductionRuleService.
type deductionRuleService struct {
	deductionRuleRepo repository.DeductionRuleRepository
	cache             *cache.CacheService
}

// NewDeductionRuleService creates a new instance of the deduction rule service.
func NewDeductionRuleService(deductionRuleRepo repository.DeductionRuleRepository, cacheService ...*cache.CacheService) DeductionRuleService {
	var cs *cache.CacheService
	if len(cacheService) > 0 {
		cs = cacheService[0]
	}
	return &deductionRuleService{
		deductionRuleRepo: deductionRuleRepo,
		cache:             cs,
	}
}

func deductionCacheKey(businessID uint) string {
	return fmt.Sprintf("deduction_rules:business:%d", businessID)
}

// CreateDeductionRule validates and creates a new deduction rule.
func (s *deductionRuleService) CreateDeductionRule(ctx context.Context, rule *domain.DeductionRule) (*domain.DeductionRule, error) {
	if err := s.deductionRuleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create deduction rule: %w", err)
	}
	s.invalidateCache(ctx, rule.BusinessID)
	return rule, nil
}

// ListByBusinessID retrieves all deduction rules for a business (cached 5 min).
func (s *deductionRuleService) ListByBusinessID(ctx context.Context, businessID uint) ([]*domain.DeductionRule, error) {
	return cache.GetOrLoad(s.cache, ctx, deductionCacheKey(businessID), 5*time.Minute, func() ([]*domain.DeductionRule, error) {
		return s.deductionRuleRepo.FindByBusinessID(ctx, businessID)
	})
}

// GetByID retrieves a specific deduction rule by ID, ensuring it belongs to the specified business.
func (s *deductionRuleService) GetByID(ctx context.Context, ruleID, businessID uint) (*domain.DeductionRule, error) {
	return s.deductionRuleRepo.FindByID(ctx, ruleID, businessID)
}

// UpdateDeductionRule updates an existing deduction rule.
func (s *deductionRuleService) UpdateDeductionRule(ctx context.Context, rule *domain.DeductionRule) (*domain.DeductionRule, error) {
	if err := s.deductionRuleRepo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update deduction rule: %w", err)
	}
	s.invalidateCache(ctx, rule.BusinessID)
	return rule, nil
}

// DeleteDeductionRule deletes a deduction rule, ensuring it belongs to the specified business.
func (s *deductionRuleService) DeleteDeductionRule(ctx context.Context, ruleID, businessID uint) error {
	if err := s.deductionRuleRepo.Delete(ctx, ruleID, businessID); err != nil {
		return err
	}
	s.invalidateCache(ctx, businessID)
	return nil
}

func (s *deductionRuleService) invalidateCache(ctx context.Context, businessID uint) {
	if s.cache != nil {
		s.cache.Invalidate(ctx, deductionCacheKey(businessID))
	}
}
