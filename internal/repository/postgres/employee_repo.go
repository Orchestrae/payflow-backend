// internal/repository/postgres/employee_repo.go
package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"gorm.io/gorm"
)

type employeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) repository.EmployeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) FindActiveByBusinessID(ctx context.Context, businessID uint) ([]domain.Employee, error) {
	var dbEmployees []Employee
	// Here is the magic of GORM's preloading for relational data.
	// It fetches employees and, for each employee, fetches their associated Cadre
	// and for each Cadre, its EarningComponents and DeductionRules.
	err := r.db.WithContext(ctx).
		Preload("Cadre.EarningComponents").
		Preload("Cadre.DeductionRules").
		Where("business_id = ? AND is_active = ?", businessID, true).
		Find(&dbEmployees).Error

	if err != nil {
		return nil, err
	}

	domainEmployees := make([]domain.Employee, len(dbEmployees))
	for i, dbEmp := range dbEmployees {
		domainEmployees[i] = *dbEmp.ToDomain()
	}

	return domainEmployees, nil
}

func (r *employeeRepository) Deactivate(ctx context.Context, employeeID uint) error {
	result := r.db.WithContext(ctx).Model(&Employee{}).Where("id = ?", employeeID).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *employeeRepository) Create(ctx context.Context, employee *domain.Employee) error {
	dbEmployee := EmployeeFromDomain(employee)
	if err := r.db.WithContext(ctx).Create(dbEmployee).Error; err != nil {
		return err
	}
	*employee = *dbEmployee.ToDomain()
	return nil
}

func (r *employeeRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&Employee{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *employeeRepository) FindAllByBusinessID(ctx context.Context, businessID uint) ([]domain.Employee, error) {
	var dbEmployees []Employee
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Find(&dbEmployees).Error
	if err != nil {
		return nil, err
	}
	domainEmployees := make([]domain.Employee, len(dbEmployees))
	for i, dbEmp := range dbEmployees {
		domainEmployees[i] = *dbEmp.ToDomain()
	}
	return domainEmployees, nil
}

func (r *employeeRepository) FindByID(ctx context.Context, id uint) (*domain.Employee, error) {
	var dbEmployee Employee
	if err := r.db.WithContext(ctx).First(&dbEmployee, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return dbEmployee.ToDomain(), nil
}

func (r *employeeRepository) Update(ctx context.Context, employee *domain.Employee) error {
	dbEmployee := EmployeeFromDomain(employee)
	result := r.db.WithContext(ctx).Model(&dbEmployee).Updates(dbEmployee)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *employeeRepository) WithTx(tx *gorm.DB) repository.EmployeeRepository {
	return NewEmployeeRepository(tx)
}
