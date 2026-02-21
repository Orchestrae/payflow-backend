// internal/repository/postgres/business_repo.go
package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"gorm.io/gorm"
)

type businessRepository struct {
	db *gorm.DB
}

func NewBusinessRepository(db *gorm.DB) repository.BusinessRepository {
	return &businessRepository{db: db}
}

// WithTx allows this repository to be used within a transaction.
func (r *businessRepository) WithTx(tx repository.Transactioner) repository.BusinessRepository {
	if txr, ok := tx.(*transactioner); ok {
		return &businessRepository{db: txr.db}
	}
	return r
}

func (r *businessRepository) Create(ctx context.Context, business *domain.Business) error {
	dbBusiness := BusinessFromDomain(business)
	if err := r.db.WithContext(ctx).Create(dbBusiness).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	*business = *dbBusiness.ToDomain()
	return nil
}

func (r *businessRepository) Update(ctx context.Context, business *domain.Business) error {
	dbBusiness := BusinessFromDomain(business)
	// We use .Model(&dbBusiness) to ensure we only update fields on an existing record
	// and trigger GORM's BeforeUpdate hooks, which automatically updates `updated_at`.
	result := r.db.WithContext(ctx).Model(&dbBusiness).Updates(dbBusiness)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *businessRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&Business{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *businessRepository) FindByID(ctx context.Context, id uint) (*domain.Business, error) {
	var dbBusiness Business
	if err := r.db.WithContext(ctx).First(&dbBusiness, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return dbBusiness.ToDomain(), nil
}

func (r *businessRepository) FindByRCNumber(ctx context.Context, rcNumber string) (*domain.Business, error) {
	var dbBusiness Business
	if err := r.db.WithContext(ctx).Where("rc_number = ?", rcNumber).First(&dbBusiness).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return dbBusiness.ToDomain(), nil
}
