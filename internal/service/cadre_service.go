package service

import (
	"context"
	"fmt"
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
	if err := s.cadreRepo.Create(ctx, cadre); err != nil {
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
