package service

import (
	"context"
	"fmt"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// LoanService manages employee loans.
type LoanService interface {
	Create(ctx context.Context, loan *domain.EmployeeLoan) (*domain.EmployeeLoan, error)
	ListByBusiness(ctx context.Context, businessID uint, page, limit int) ([]*domain.EmployeeLoan, int, error)
	GetActiveByEmployee(ctx context.Context, employeeID uint) ([]*domain.EmployeeLoan, error)
	Cancel(ctx context.Context, loanID, businessID uint) error
}

type loanService struct {
	loanRepo repository.LoanRepository
}

// NewLoanService creates a new loan service.
func NewLoanService(loanRepo repository.LoanRepository) LoanService {
	return &loanService{loanRepo: loanRepo}
}

func (s *loanService) Create(ctx context.Context, loan *domain.EmployeeLoan) (*domain.EmployeeLoan, error) {
	loan.RemainingBalance = loan.LoanAmount
	loan.TotalRepaid = 0
	loan.Status = "active"
	if err := s.loanRepo.Create(ctx, loan); err != nil {
		return nil, fmt.Errorf("failed to create loan: %w", err)
	}
	return loan, nil
}

func (s *loanService) ListByBusiness(ctx context.Context, businessID uint, page, limit int) ([]*domain.EmployeeLoan, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.loanRepo.FindByBusinessID(ctx, businessID, page, limit)
}

func (s *loanService) GetActiveByEmployee(ctx context.Context, employeeID uint) ([]*domain.EmployeeLoan, error) {
	return s.loanRepo.FindActiveByEmployeeID(ctx, employeeID)
}

func (s *loanService) Cancel(ctx context.Context, loanID, businessID uint) error {
	loan, err := s.loanRepo.FindByID(ctx, loanID)
	if err != nil {
		return domain.ErrNotFound
	}
	if loan.BusinessID != businessID {
		return domain.ErrForbidden
	}
	if loan.Status != "active" {
		return fmt.Errorf("%w: can only cancel active loans", domain.ErrValidationFailed)
	}
	loan.Status = "cancelled"
	return s.loanRepo.Update(ctx, loan)
}
