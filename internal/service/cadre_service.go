package service

import (
	"context"
	"fmt"
	"log/slog"
	"payflow/internal/domain"
	"payflow/internal/repository"
)

// CadreService defines the interface for cadre-related business logic.
type CadreService interface {
	CreateCadre(ctx context.Context, cadre *domain.Cadre) (*domain.Cadre, error)
	ListByBusinessID(ctx context.Context, businessID uint) ([]*domain.Cadre, error)
	GetByID(ctx context.Context, cadreID, businessID uint) (*domain.Cadre, error)
	UpdateCadre(ctx context.Context, cadre *domain.Cadre) (*domain.Cadre, error)
	DeleteCadre(ctx context.Context, cadreID, businessID uint) error
}

// cadreService is the concrete implementation of the CadreService.
type cadreService struct {
	cadreRepo repository.CadreRepository
}

// NewCadreService creates a new instance of the cadre service.
func NewCadreService(cadreRepo repository.CadreRepository) CadreService {
	return &cadreService{
		cadreRepo: cadreRepo,
	}
}

// CreateCadre validates and creates a new cadre.
func (s *cadreService) CreateCadre(ctx context.Context, cadre *domain.Cadre) (*domain.Cadre, error) {
	// Validate cadre name is unique for this business
	isUnique, err := s.cadreRepo.IsCadreNameUnique(ctx, *cadre)
	if err != nil {
		slog.Error("Error checking cadre name uniqueness", "error", err)
		return nil, fmt.Errorf("failed to check cadre name uniqueness: %w", err)
	}
	if !isUnique {
		slog.Error("Cadre name already exists for this business", "name", cadre.Name, "business_id", cadre.BusinessID)
		// Return domain.ErrConflict so it's properly handled by RespondWithError
		return nil, domain.ErrConflict
	}
	//todo for a cadre name that is repeated, we can seamply chick if the earning component name does not already exist
	// in the existing cadre, if not, we can allow the creation of the cadre
	//else we can have an independaent update cadre feature
	// Validate earning component names are unique within this cadre
	if err := validateEarningComponentNamesUnique(cadre.EarningComponents); err != nil {
		slog.Error("Earning component names are not unique", "error", err)
		// Return domain.ErrValidationFailed so it's properly handled by RespondWithError
		return nil, domain.ErrValidationFailed
	}

	// TODO: Apply deduction rules from cadre.DeductionRules if needed

	// Create the cadre
	if err := s.cadreRepo.Create(ctx, cadre); err != nil {
		slog.Error("Failed to create cadre", "error", err, "business_id", cadre.BusinessID, "name", cadre.Name)
		return nil, fmt.Errorf("failed to create cadre: %w", err)
	}

	return cadre, nil
}

// ListByBusinessID retrieves all cadres for a business.
func (s *cadreService) ListByBusinessID(ctx context.Context, businessID uint) ([]*domain.Cadre, error) {
	return s.cadreRepo.FindByBusinessID(ctx, businessID)
}

// GetByID retrieves a specific cadre by ID, ensuring it belongs to the specified business.
func (s *cadreService) GetByID(ctx context.Context, cadreID, businessID uint) (*domain.Cadre, error) {
	return s.cadreRepo.FindByID(ctx, cadreID, businessID)
}

// UpdateCadre updates an existing cadre.
func (s *cadreService) UpdateCadre(ctx context.Context, cadre *domain.Cadre) (*domain.Cadre, error) {
	if err := s.cadreRepo.Update(ctx, cadre); err != nil {
		return nil, fmt.Errorf("failed to update cadre: %w", err)
	}
	return cadre, nil
}

// DeleteCadre deletes a cadre, ensuring it belongs to the specified business.
func (s *cadreService) DeleteCadre(ctx context.Context, cadreID, businessID uint) error {
	return s.cadreRepo.Delete(ctx, cadreID, businessID)
}

func validateEarningComponentNamesUnique(
	components []domain.EarningComponent,
) error {

	seen := make(map[string]struct{})

	for _, c := range components {
		if _, exists := seen[c.Name]; exists {
			return fmt.Errorf("earning component name '%s' is duplicated", c.Name)
		}
		seen[c.Name] = struct{}{}
	}

	return nil
}
