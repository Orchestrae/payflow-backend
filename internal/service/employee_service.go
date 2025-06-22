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
	cadre, err := s.cadreRepo.FindByID(ctx, emp.CadreID)
	if err != nil {
		if err == domain.ErrNotFound {
			log.Ctx(ctx).Warn().Uint("cadreID", emp.CadreID).Msg("Attempt to create employee with non-existent cadre")
			return nil, fmt.Errorf("%w: cadre with ID %d not found", domain.ErrValidationFailed, emp.CadreID)
		}
		return nil, err // Internal error
	}

	//validate that the cadre belongs to the same business as the employee
	// Security Check: Ensure the cadre belongs to the same business as the employee.

	if cadre.BusinessID != emp.BusinessID {
		log.Ctx(ctx).Warn().Uint("employeeBusinessID", emp.BusinessID).Uint("cadreBusinessID", cadre.BusinessID).Msg("Attempt to assign cadre from another business")
		return nil, domain.ErrForbidden
	}
	// validate that email is unique within the business using IsEmailExistByBusiness
	var exists bool
	if exists, err = s.employeeRepo.IsEmailExistByBusiness(ctx, emp.Email, emp.BusinessID); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to check if email exists in business")
		return nil, err // Internal error
	}
	if exists {
		log.Ctx(ctx).Warn().Str("email", emp.Email).Uint("businessID", emp.BusinessID).Msg("Attempt to create employee with duplicate email")
		return nil, fmt.Errorf("%w: email %s already exists for this business", domain.ErrValidationFailed, emp.Email)
	}

	// The repository's Create method will handle potential conflicts (e.g., duplicate email).
	if err := s.employeeRepo.Create(ctx, emp); err != nil {
		return nil, err
	}

	// Return the employee with the generated ID and timestamps.
	return emp, nil
}

// ListByBusinessID retrieves all employees for a given business.
func (s *employeeService) ListByBusinessID(ctx context.Context, businessID uint) ([]domain.Employee, error) {
	employees, err := s.employeeRepo.FindAllByBusinessID(ctx, businessID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Uint("businessID", businessID).Msg("Failed to list employees by business ID")
		return nil, err
	}
	return employees, nil
}

// GetByID retrieves a single employee, ensuring they belong to the specified business.
func (s *employeeService) GetByID(ctx context.Context, employeeID, businessID uint) (*domain.Employee, error) {
	emp, err := s.employeeRepo.FindByID(ctx, employeeID)
	if err != nil {
		return nil, err // Will return domain.ErrNotFound if not found.
	}

	// Security Check: Ensure the requested employee belongs to the user's business.
	if emp.BusinessID != businessID {
		log.Ctx(ctx).Warn().Uint("employeeID", employeeID).Uint("requestingBusinessID", businessID).Msg("Forbidden attempt to access employee from another business")
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
		cadre, err := s.cadreRepo.FindByID(ctx, emp.CadreID)
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

	// 2. Perform the deactivation.
	if err := s.employeeRepo.Deactivate(ctx, emp.ID); err != nil {
		log.Ctx(ctx).Error().Err(err).Uint("employeeID", emp.ID).Msg("Failed to deactivate employee in repository")
		return err
	}

	log.Ctx(ctx).Info().Uint("employeeID", emp.ID).Msg("Employee successfully deactivated")
	return nil
}
