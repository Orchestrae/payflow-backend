package postgres

import (
	"context"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

type loanRepository struct {
	db *gorm.DB
}

func NewLoanRepository(db *gorm.DB) repository.LoanRepository {
	return &loanRepository{db: db}
}

func (r *loanRepository) Create(ctx context.Context, loan *domain.EmployeeLoan) error {
	return r.db.WithContext(ctx).Create(loan).Error
}

func (r *loanRepository) FindByID(ctx context.Context, id uint) (*domain.EmployeeLoan, error) {
	var loan domain.EmployeeLoan
	if err := r.db.WithContext(ctx).Preload("Employee").First(&loan, id).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return &loan, nil
}

func (r *loanRepository) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.EmployeeLoan, int, error) {
	var loans []*domain.EmployeeLoan
	var total int64

	r.db.WithContext(ctx).Model(&domain.EmployeeLoan{}).Where("business_id = ?", businessID).Count(&total)

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).Preload("Employee").Where("business_id = ?", businessID).
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&loans).Error

	return loans, int(total), err
}

func (r *loanRepository) FindActiveByEmployeeID(ctx context.Context, employeeID uint) ([]*domain.EmployeeLoan, error) {
	var loans []*domain.EmployeeLoan
	err := r.db.WithContext(ctx).Where("employee_id = ? AND status = ?", employeeID, "active").Find(&loans).Error
	return loans, err
}

func (r *loanRepository) Update(ctx context.Context, loan *domain.EmployeeLoan) error {
	return r.db.WithContext(ctx).Save(loan).Error
}
