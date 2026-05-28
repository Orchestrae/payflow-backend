package service

import (
	"context"
	"fmt"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// BusinessService defines business-level operations.
type BusinessService interface {
	GetSettings(ctx context.Context, businessID uint) (*domain.Business, error)
	UpdateSettings(ctx context.Context, businessID uint, updates map[string]interface{}) (*domain.Business, error)
}

type businessService struct {
	businessRepo repository.BusinessRepository
}

// NewBusinessService creates a new business service.
func NewBusinessService(businessRepo repository.BusinessRepository) BusinessService {
	return &businessService{businessRepo: businessRepo}
}

func (s *businessService) GetSettings(ctx context.Context, businessID uint) (*domain.Business, error) {
	business, err := s.businessRepo.FindByID(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to get business: %w", err)
	}
	return business, nil
}

func (s *businessService) UpdateSettings(ctx context.Context, businessID uint, updates map[string]interface{}) (*domain.Business, error) {
	business, err := s.businessRepo.FindByID(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("business not found: %w", err)
	}

	// Apply updates from the map
	if v, ok := updates["pension_enabled"]; ok {
		business.PensionEnabled = v.(bool)
	}
	if v, ok := updates["nhf_enabled"]; ok {
		business.NHFEnabled = v.(bool)
	}
	if v, ok := updates["nsitf_enabled"]; ok {
		business.NSITFEnabled = v.(bool)
	}
	if v, ok := updates["paye_enabled"]; ok {
		business.PAYEEnabled = v.(bool)
	}
	if v, ok := updates["payroll_requires_approval"]; ok {
		business.PayrollRequiresApproval = v.(bool)
	}
	if v, ok := updates["payroll_auto_process"]; ok {
		business.PayrollAutoProcess = v.(bool)
	}

	if err := s.businessRepo.Update(ctx, business); err != nil {
		return nil, fmt.Errorf("failed to update business settings: %w", err)
	}

	return business, nil
}
