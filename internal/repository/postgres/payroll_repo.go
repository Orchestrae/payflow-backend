package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"gorm.io/gorm"
)

type PayrollRepository struct {
	db *gorm.DB
}

func NewPayrollRepository(db *gorm.DB) *PayrollRepository {
	return &PayrollRepository{db: db}
}

func (r *PayrollRepository) Create(ctx context.Context, run *domain.PayrollRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *PayrollRepository) GetByID(ctx context.Context, id uint) (*domain.PayrollRun, error) {
	var run domain.PayrollRun
	err := r.db.WithContext(ctx).
		Preload("Entries").
		Preload("Entries.Employee").
		Preload("Entries.Details").
		First(&run, id).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *PayrollRepository) Update(ctx context.Context, run *domain.PayrollRun) error {
	return r.db.WithContext(ctx).Save(run).Error
}

func (r *PayrollRepository) ListByBusinessID(ctx context.Context, businessID uint) ([]domain.PayrollRun, error) {
	var runs []domain.PayrollRun
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Preload("Entries").
		Preload("Entries.Employee").
		Preload("Entries.Details").
		Order("created_at DESC").
		Find(&runs).Error
	if err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *PayrollRepository) UpdateStatus(ctx context.Context, id uint, status domain.PayrollStatus) error {
	return r.db.WithContext(ctx).
		Model(&domain.PayrollRun{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *PayrollRepository) UpdatePaymentReference(ctx context.Context, id uint, reference string) error {
	return r.db.WithContext(ctx).
		Model(&domain.PayrollRun{}).
		Where("id = ?", id).
		Update("payment_reference", reference).Error
}

func (r *PayrollRepository) CreateRun(ctx context.Context, tx *gorm.DB, run *domain.PayrollRun) error {
	return r.Create(ctx, run)
}

func (r *PayrollRepository) Delete(ctx context.Context, id uint, businessID uint) error {
	result := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Delete(&domain.PayrollRun{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PayrollRepository) FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.PayrollRun, error) {
	var runs []domain.PayrollRun
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Preload("Entries").
		Preload("Entries.Employee").
		Preload("Entries.Details").
		Order("created_at DESC").
		Find(&runs).Error
	if err != nil {
		return nil, err
	}

	ptrRuns := make([]*domain.PayrollRun, len(runs))
	for i, run := range runs {
		r := run
		ptrRuns[i] = &r
	}
	return ptrRuns, nil
}

func (r *PayrollRepository) FindByID(ctx context.Context, id uint, businessID uint) (*domain.PayrollRun, error) {
	var run domain.PayrollRun
	query := r.db.WithContext(ctx).
		Preload("Entries").
		Preload("Entries.Employee").
		Preload("Entries.Details").
		Where("id = ?", id)

	// Support System Access (0)
	if businessID != 0 {
		query = query.Where("business_id = ?", businessID)
	}

	err := query.First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *PayrollRepository) WithTx(tx repository.Transactioner) repository.PayrollRepository {
	// Cast to *transactioner (private struct in this package)
	if txr, ok := tx.(*transactioner); ok {
		return &PayrollRepository{db: txr.db}
	}
	return r
}
