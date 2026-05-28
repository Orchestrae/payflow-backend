package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// WalletModel is the GORM model for business_wallets
type WalletModel struct {
	gorm.Model
	BusinessID              uint
	Balance                 int64 `gorm:"default:0"`
	LockedBalance           int64 `gorm:"default:0"`
	Currency                string `gorm:"size:10;default:'NGN'"`
	BalanceUpdatedAt        *time.Time
	VirtualAccountNumber    string `gorm:"size:20;uniqueIndex"`
	VirtualAccountBankCode  string `gorm:"size:10"`
	VirtualAccountBankName  string `gorm:"size:255"`
	VirtualAccountReference string `gorm:"size:100;uniqueIndex"`
	VirtualAccountUniqueID  string `gorm:"size:100"`
	VirtualAccountStatus    string `gorm:"size:20;default:'active'"`
	Provider                string `gorm:"size:20"`
	ProviderMetadata        *string `gorm:"type:jsonb"` // JSON string stored in database
}

// TableName specifies the table name for GORM
func (WalletModel) TableName() string {
	return "business_wallets"
}

// ToDomain converts the GORM model to domain entity
func (w *WalletModel) ToDomain() *domain.BusinessWallet {
	wallet := &domain.BusinessWallet{
		Model: domain.Model{
			ID:        w.Model.ID,
			CreatedAt: w.Model.CreatedAt,
			UpdatedAt: w.Model.UpdatedAt,
			DeletedAt: w.Model.DeletedAt,
		},
		BusinessID:              w.BusinessID,
		Balance:                 w.Balance,
		LockedBalance:           w.LockedBalance,
		Currency:                w.Currency,
		BalanceUpdatedAt:        w.BalanceUpdatedAt,
		VirtualAccountNumber:    w.VirtualAccountNumber,
		VirtualAccountBankCode:  w.VirtualAccountBankCode,
		VirtualAccountBankName:  w.VirtualAccountBankName,
		VirtualAccountReference: w.VirtualAccountReference,
		VirtualAccountUniqueID:  w.VirtualAccountUniqueID,
		VirtualAccountStatus:    w.VirtualAccountStatus,
		Provider:                domain.ProviderName(w.Provider),
	}

	// Parse provider metadata if available
	if w.ProviderMetadata != nil && *w.ProviderMetadata != "" {
		wallet.ProviderMetadata = *w.ProviderMetadata
	}

	return wallet
}

// WalletModelFromDomain converts domain entity to GORM model
func WalletModelFromDomain(w *domain.BusinessWallet) *WalletModel {
	model := &WalletModel{
		Model: gorm.Model{
			ID:        w.Model.ID,
			CreatedAt: w.Model.CreatedAt,
			UpdatedAt: w.Model.UpdatedAt,
			DeletedAt: w.Model.DeletedAt,
		},
		BusinessID:              w.BusinessID,
		Balance:                 w.Balance,
		LockedBalance:           w.LockedBalance,
		Currency:                w.Currency,
		BalanceUpdatedAt:        w.BalanceUpdatedAt,
		VirtualAccountNumber:    w.VirtualAccountNumber,
		VirtualAccountBankCode:  w.VirtualAccountBankCode,
		VirtualAccountBankName:  w.VirtualAccountBankName,
		VirtualAccountReference: w.VirtualAccountReference,
		VirtualAccountUniqueID:  w.VirtualAccountUniqueID,
		VirtualAccountStatus:    w.VirtualAccountStatus,
		Provider:                string(w.Provider),
	}

	// Store provider metadata as JSON string
	if w.ProviderMetadata != "" {
		// Validate JSON first
		var jsonMap map[string]interface{}
		if err := json.Unmarshal([]byte(w.ProviderMetadata), &jsonMap); err == nil {
			metadataStr := w.ProviderMetadata
			model.ProviderMetadata = &metadataStr
		}
	}

	return model
}

// walletRepository implements repository.WalletRepository
type walletRepository struct {
	db *gorm.DB
}

// NewWalletRepository creates a new wallet repository
func NewWalletRepository(db *gorm.DB) repository.WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) Create(ctx context.Context, wallet *domain.BusinessWallet) error {
	model := WalletModelFromDomain(wallet)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrConflict
		}
		return DBErrToDomainErr(err)
	}
	*wallet = *model.ToDomain()
	return nil
}

func (r *walletRepository) FindByBusinessID(ctx context.Context, businessID uint) (*domain.BusinessWallet, error) {
	var model WalletModel
	if err := r.db.WithContext(ctx).Where("business_id = ?", businessID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, DBErrToDomainErr(err)
	}
	return model.ToDomain(), nil
}

func (r *walletRepository) FindByAccountReference(ctx context.Context, accountReference string) (*domain.BusinessWallet, error) {
	var model WalletModel
	if err := r.db.WithContext(ctx).Where("virtual_account_reference = ?", accountReference).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, DBErrToDomainErr(err)
	}
	return model.ToDomain(), nil
}

func (r *walletRepository) Update(ctx context.Context, wallet *domain.BusinessWallet) error {
	model := WalletModelFromDomain(wallet)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return DBErrToDomainErr(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	*wallet = *model.ToDomain()
	return nil
}

func (r *walletRepository) IncrementBalance(ctx context.Context, businessID uint, amount int64) (*domain.BusinessWallet, error) {
	result := r.db.WithContext(ctx).Exec(
		"UPDATE business_wallets SET balance = balance + ?, balance_updated_at = NOW(), updated_at = NOW() WHERE business_id = ? AND deleted_at IS NULL",
		amount, businessID,
	)
	if result.Error != nil {
		return nil, DBErrToDomainErr(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	return r.FindByBusinessID(ctx, businessID)
}

func (r *walletRepository) DecrementBalanceAndLocked(ctx context.Context, businessID uint, balanceAmount int64, lockedAmount int64) (*domain.BusinessWallet, error) {
	result := r.db.WithContext(ctx).Exec(
		"UPDATE business_wallets SET balance = balance - ?, locked_balance = GREATEST(locked_balance - ?, 0), balance_updated_at = NOW(), updated_at = NOW() WHERE business_id = ? AND deleted_at IS NULL AND balance >= ?",
		balanceAmount, lockedAmount, businessID, balanceAmount,
	)
	if result.Error != nil {
		return nil, DBErrToDomainErr(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("insufficient balance for withdrawal")
	}
	return r.FindByBusinessID(ctx, businessID)
}

func (r *walletRepository) IncrementLocked(ctx context.Context, businessID uint, amount int64) (*domain.BusinessWallet, error) {
	result := r.db.WithContext(ctx).Exec(
		"UPDATE business_wallets SET locked_balance = locked_balance + ?, updated_at = NOW() WHERE business_id = ? AND deleted_at IS NULL AND (balance - locked_balance) >= ?",
		amount, businessID, amount,
	)
	if result.Error != nil {
		return nil, DBErrToDomainErr(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("insufficient available balance")
	}
	return r.FindByBusinessID(ctx, businessID)
}

func (r *walletRepository) DecrementLocked(ctx context.Context, businessID uint, amount int64) (*domain.BusinessWallet, error) {
	result := r.db.WithContext(ctx).Exec(
		"UPDATE business_wallets SET locked_balance = GREATEST(locked_balance - ?, 0), updated_at = NOW() WHERE business_id = ? AND deleted_at IS NULL",
		amount, businessID,
	)
	if result.Error != nil {
		return nil, DBErrToDomainErr(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	return r.FindByBusinessID(ctx, businessID)
}

func (r *walletRepository) WithTx(tx *gorm.DB) repository.WalletRepository {
	return &walletRepository{db: tx}
}
