// internal/service/employee_service.go
package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"github.com/rs/zerolog/log"
)

// EmployeeService defines the interface for employee-related business logic.
type EmployeeService interface {
	CreateEmployee(ctx context.Context, emp *domain.Employee) (*domain.Employee, error)
	ListByBusinessID(ctx context.Context, businessID uint) ([]domain.Employee, error)
	GetByID(ctx context.Context, employeeID, businessID uint) (*domain.Employee, error)
	UpdateEmployee(ctx context.Context, emp *domain.Employee) (*domain.Employee, error)
	DeactivateEmployee(ctx context.Context, employeeID, businessID uint) error
}

// employeeService is the concrete implementation of the EmployeeService.
type employeeService struct {
	employeeRepo repository.EmployeeRepository
	cadreRepo    repository.CadreRepository
}

// NewEmployeeService creates a new instance of the employee service.
func NewEmployeeService(empRepo repository.EmployeeRepository, cadreRepo repository.CadreRepository) EmployeeService {
	return &employeeService{
		employeeRepo: empRepo,
		cadreRepo:    cadreRepo,
	}
}

// CreateEmployee validates and creates a new employee.
func (s *employeeService) CreateEmployee(ctx context.Context, emp *domain.Employee) (*domain.Employee, error) {
	// Business Rule: Ensure the assigned cadre exists and belongs to the same business.
	// We pass emp.BusinessID to enforce tenancy.
	cadre, err := s.cadreRepo.FindByID(ctx, emp.CadreID, emp.BusinessID)
	if err != nil {
		if err == domain.ErrNotFound {
			log.Ctx(ctx).Warn().Uint("cadreID", emp.CadreID).Msg("Attempt to create employee with non-existent or forbidden cadre")
			return nil, fmt.Errorf("%w: cadre with ID %d not found for business", domain.ErrValidationFailed, emp.CadreID)
		}
		log.Error().Err(err).Uint("cadreID", emp.CadreID).Msg("Error retrieving cadre for employee creation")
		return nil, err // Internal error
	}

	// Double check (redundant if repo handles it, but safe)
	if cadre.BusinessID != emp.BusinessID {
		return nil, domain.ErrForbidden
	}

	// TODO: This email uniqueness check loads ALL employees for the business, which is inefficient
	// for large teams. Replace with a dedicated repository method like FindByEmail(ctx, businessID, email)
	// or a unique DB constraint on (business_id, email) once the repository supports it.
	employees, err := s.employeeRepo.FindByBusinessID(ctx, emp.BusinessID)
	if err != nil {
		return nil, err
	}
	for _, e := range employees {
		if e.Email == emp.Email {
			log.Ctx(ctx).Warn().Str("email", emp.Email).Uint("businessID", emp.BusinessID).Msg("Attempt to create employee with duplicate email")
			return nil, fmt.Errorf("%w: email %s already exists for this business", domain.ErrValidationFailed, emp.Email)
		}
	}

	// The repository's Create method will handle potential conflicts if DB constraints exist.
	if err := s.employeeRepo.Create(ctx, emp); err != nil {
		return nil, err
	}

	// Return the employee with the generated ID and timestamps.
	return emp, nil
}

// ListByBusinessID retrieves all employees for a given business.
func (s *employeeService) ListByBusinessID(ctx context.Context, businessID uint) ([]domain.Employee, error) {
	// FindByBusinessID returns []*domain.Employee
	employeesPtrs, err := s.employeeRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Uint("businessID", businessID).Msg("Failed to list employees by business ID")
		return nil, err
	}

	// Convert []*domain.Employee to []domain.Employee
	employees := make([]domain.Employee, len(employeesPtrs))
	for i, v := range employeesPtrs {
		employees[i] = *v
	}

	return employees, nil
}

// GetByID retrieves a single employee, ensuring they belong to the specified business.
func (s *employeeService) GetByID(ctx context.Context, employeeID, businessID uint) (*domain.Employee, error) {
	// Pass businessID to repo to enforce tenancy finding
	emp, err := s.employeeRepo.FindByID(ctx, employeeID, businessID)
	if err != nil {
		return nil, err // Will return domain.ErrNotFound if not found.
	}

	// Double check implied by repo call
	if emp.BusinessID != businessID {
		return nil, domain.ErrForbidden
	}

	return emp, nil
}

// UpdateEmployee validates and updates an existing employee's details.
func (s *employeeService) UpdateEmployee(ctx context.Context, emp *domain.Employee) (*domain.Employee, error) {
	// 1. First, verify the employee exists and belongs to the correct business.
	existingEmp, err := s.GetByID(ctx, emp.ID, emp.BusinessID)
	if err != nil {
		return nil, err // This will handle both Not Found and Forbidden errors.
	}

	// 2. Business Rule: If the cadre is being changed, validate the new cadre.
	if emp.CadreID != existingEmp.CadreID {
		cadre, err := s.cadreRepo.FindByID(ctx, emp.CadreID, emp.BusinessID)
		if err != nil {
			if err == domain.ErrNotFound {
				return nil, fmt.Errorf("%w: new cadre with ID %d not found", domain.ErrValidationFailed, emp.CadreID)
			}
			return nil, err
		}
		if cadre.BusinessID != emp.BusinessID {
			return nil, domain.ErrForbidden
		}
	}

	// 3. Update the record in the database.
	// Assuming Update handles field replacement.
	if err := s.employeeRepo.Update(ctx, emp); err != nil {
		return nil, err
	}

	return emp, nil
}

// DeactivateEmployee marks an employee as inactive. They are not deleted from the database.
func (s *employeeService) DeactivateEmployee(ctx context.Context, employeeID, businessID uint) error {
	// 1. Verify the employee exists and belongs to the business before deactivating.
	emp, err := s.GetByID(ctx, employeeID, businessID)
	if err != nil {
		return err
	}

	// 2. Perform the deactivation via Update.
	emp.IsActive = false
	if err := s.employeeRepo.Update(ctx, emp); err != nil {
		log.Ctx(ctx).Error().Err(err).Uint("employeeID", emp.ID).Msg("Failed to deactivate employee in repository")
		return err
	}

	log.Ctx(ctx).Info().Uint("employeeID", emp.ID).Msg("Employee successfully deactivated")
	return nil
}
