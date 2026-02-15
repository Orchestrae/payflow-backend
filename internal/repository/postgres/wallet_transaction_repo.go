package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

type walletTransactionRepository struct {
	db *gorm.DB
}

// NewWalletTransactionRepository creates a new wallet transaction repository.
func NewWalletTransactionRepository(db *gorm.DB) repository.WalletTransactionRepository {
	return &walletTransactionRepository{db: db}
}

func (r *walletTransactionRepository) Create(ctx context.Context, tx *domain.WalletTransaction) error {
	if err := r.db.WithContext(ctx).Create(tx).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrConflict
		}
		return DBErrToDomainErr(err)
	}
	return nil
}

func (r *walletTransactionRepository) FindByID(ctx context.Context, id uint) (*domain.WalletTransaction, error) {
	var tx domain.WalletTransaction
	if err := r.db.WithContext(ctx).First(&tx, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, DBErrToDomainErr(err)
	}
	return &tx, nil
}

func (r *walletTransactionRepository) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.WalletTransaction, int, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&domain.WalletTransaction{}).Where("business_id = ?", businessID).Count(&total).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	var models []domain.WalletTransaction
	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).Where("business_id = ?", businessID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	result := make([]*domain.WalletTransaction, len(models))
	for i := range models {
		result[i] = &models[i]
	}
	return result, int(total), nil
}

func (r *walletTransactionRepository) FindByReference(ctx context.Context, reference string) (*domain.WalletTransaction, error) {
	var tx domain.WalletTransaction
	if err := r.db.WithContext(ctx).Where("reference = ?", reference).First(&tx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, DBErrToDomainErr(err)
	}
	return &tx, nil
}

func (r *walletTransactionRepository) Update(ctx context.Context, tx *domain.WalletTransaction) error {
	result := r.db.WithContext(ctx).Save(tx)
	if result.Error != nil {
		return DBErrToDomainErr(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *walletTransactionRepository) WithTx(tx *gorm.DB) repository.WalletTransactionRepository {
	return &walletTransactionRepository{db: tx}
}
