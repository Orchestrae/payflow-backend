// internal/repository/repository.go
package repository

import (
	"context"
	"payflow/internal/domain"

	"gorm.io/gorm"
)

// Transactioner defines an interface for managing database transactions.
type Transactioner interface {
	Begin(ctx context.Context) *gorm.DB
	Commit(tx *gorm.DB) error
	Rollback(tx *gorm.DB)
}

// WithTx is an interface for repositories that can operate within a transaction.
type WithTx interface {
	WithTx(tx *gorm.DB) any // Returns a new repository instance with the transaction
}

// BaseRepository defines common methods for all repositories.
type BaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	FindByID(ctx context.Context, id uint) (*T, error)
	Delete(ctx context.Context, id uint) error
}

// UserRepository defines the contract for user data operations.
type UserRepository interface {
	BaseRepository[domain.User]
	WithTx(tx *gorm.DB) UserRepository
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindApproversByBusinessID(ctx context.Context, businessID uint) ([]domain.User, error)
	FindOperatorsByBusinessID(ctx context.Context, businessID uint) ([]domain.User, error)
}

// BusinessRepository defines the contract for business data operations.
type BusinessRepository interface {
	BaseRepository[domain.Business]
	WithTx(tx *gorm.DB) BusinessRepository
}

// CadreRepository defines the contract for cadre data operations.
type CadreRepository interface {
	BaseRepository[domain.Cadre]
	WithTx(tx *gorm.DB) CadreRepository
	FindAllByBusinessID(ctx context.Context, businessID uint) ([]domain.Cadre, error)
	FindByID(ctx context.Context, cadreID uint) (*domain.Cadre, error)
}

// EmployeeRepository defines the contract for employee data operations.
type EmployeeRepository interface {
	BaseRepository[domain.Employee]
	WithTx(tx *gorm.DB) EmployeeRepository
	FindAllByBusinessID(ctx context.Context, businessID uint) ([]domain.Employee, error)
	FindActiveByBusinessID(ctx context.Context, businessID uint) ([]domain.Employee, error)
	Create(ctx context.Context, emp *domain.Employee) error
	FindByID(ctx context.Context, employeeID uint) (*domain.Employee, error)
	Update(ctx context.Context, emp *domain.Employee) error
	Deactivate(ctx context.Context, employeeID uint) error
}

// PayrollRepository defines the contract for payroll data operations.
type PayrollRepository interface {
	BaseRepository[domain.PayrollRun]
	WithTx(tx *gorm.DB) PayrollRepository
	FindAllByBusinessID(ctx context.Context, businessID uint) ([]domain.PayrollRun, error)
}

// DeductionRuleRepository defines the contract for deduction rule data operations.
type DeductionRuleRepository interface {
	BaseRepository[domain.DeductionRule]
	WithTx(tx *gorm.DB) DeductionRuleRepository
	FindAllByBusinessID(ctx context.Context, businessID uint) ([]domain.DeductionRule, error)
}
