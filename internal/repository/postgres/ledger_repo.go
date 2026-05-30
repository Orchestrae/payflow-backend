package postgres

import (
	"context"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

type ledgerRepository struct {
	db *gorm.DB
}

func NewLedgerRepository(db *gorm.DB) repository.LedgerRepository {
	return &ledgerRepository{db: db}
}

func (r *ledgerRepository) CreatePair(ctx context.Context, debit, credit *domain.LedgerEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(debit).Error; err != nil {
			return err
		}
		return tx.Create(credit).Error
	})
}

func (r *ledgerRepository) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.LedgerEntry, int, error) {
	var entries []*domain.LedgerEntry
	var total int64

	r.db.WithContext(ctx).Model(&domain.LedgerEntry{}).Where("business_id = ?", businessID).Count(&total)

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).Where("business_id = ?", businessID).
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&entries).Error

	return entries, int(total), err
}

func (r *ledgerRepository) GetBalanceByAccount(ctx context.Context, businessID uint, accountType domain.AccountType) (int64, error) {
	var creditSum, debitSum int64

	r.db.WithContext(ctx).Model(&domain.LedgerEntry{}).
		Where("business_id = ? AND account_type = ? AND entry_type = 'credit'", businessID, accountType).
		Select("COALESCE(SUM(amount), 0)").Scan(&creditSum)

	r.db.WithContext(ctx).Model(&domain.LedgerEntry{}).
		Where("business_id = ? AND account_type = ? AND entry_type = 'debit'", businessID, accountType).
		Select("COALESCE(SUM(amount), 0)").Scan(&debitSum)

	// For wallet (asset): balance = debits - credits (money IN is debit to wallet)
	// For external (liability): balance = credits - debits
	if accountType == domain.AccountWallet {
		return creditSum - debitSum, nil // Credits increase wallet, debits decrease
	}
	return debitSum - creditSum, nil
}

func (r *ledgerRepository) Reconcile(ctx context.Context, businessID uint) (int64, int64, int64, error) {
	var totalCredits, totalDebits int64

	r.db.WithContext(ctx).Model(&domain.LedgerEntry{}).
		Where("business_id = ? AND account_type = 'wallet'", businessID).
		Select("COALESCE(SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE 0 END), 0)").
		Scan(&totalCredits)

	r.db.WithContext(ctx).Model(&domain.LedgerEntry{}).
		Where("business_id = ? AND account_type = 'wallet'", businessID).
		Select("COALESCE(SUM(CASE WHEN entry_type = 'debit' THEN amount ELSE 0 END), 0)").
		Scan(&totalDebits)

	ledgerBalance := totalCredits - totalDebits
	return totalCredits, totalDebits, ledgerBalance, nil
}
