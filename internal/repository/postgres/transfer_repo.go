package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// TransferModel is the GORM model for transfers
type TransferModel struct {
	gorm.Model
	BusinessID             uint
	Reference              string `gorm:"size:100;uniqueIndex"`
	Amount                 string `gorm:"size:20"`
	Currency               string `gorm:"size:10;default:'NGN'"`
	Narration              string `gorm:"size:500"`
	RecipientBankCode      string `gorm:"size:10"`
	RecipientAccountNumber string `gorm:"size:20;index"`
	RecipientAccountName   string `gorm:"size:255"`
	Provider               string `gorm:"size:20"`
	Status                 string `gorm:"size:20;default:'pending'"`
	TransactionID          string `gorm:"size:100"`
	ProviderStatus         string `gorm:"size:50"`
	ProviderMessage        string `gorm:"size:500"`
	Fee                    string `gorm:"size:20"`
	ProcessedAt            *time.Time
	ProcessingError        *string `gorm:"size:1000"`
}

// TableName specifies the table name for GORM
func (TransferModel) TableName() string {
	return "transfers"
}

// ToDomain converts the GORM model to domain entity
func (t *TransferModel) ToDomain() *domain.Transfer {
	return &domain.Transfer{
		Model: domain.Model{
			ID:        t.Model.ID,
			CreatedAt: t.Model.CreatedAt,
			UpdatedAt: t.Model.UpdatedAt,
			DeletedAt: t.Model.DeletedAt,
		},
		BusinessID:             t.BusinessID,
		Reference:              t.Reference,
		Amount:                 t.Amount,
		Currency:               t.Currency,
		Narration:              t.Narration,
		RecipientBankCode:      t.RecipientBankCode,
		RecipientAccountNumber: t.RecipientAccountNumber,
		RecipientAccountName:   t.RecipientAccountName,
		Provider:               t.Provider,
		Status:                 t.Status,
		TransactionID:          t.TransactionID,
		ProviderStatus:         t.ProviderStatus,
		ProviderMessage:        t.ProviderMessage,
		Fee:                    t.Fee,
		ProcessedAt:            t.ProcessedAt,
		ProcessingError:        t.ProcessingError,
	}
}

// TransferModelFromDomain converts domain entity to GORM model
func TransferModelFromDomain(t *domain.Transfer) *TransferModel {
	return &TransferModel{
		Model: gorm.Model{
			ID:        t.Model.ID,
			CreatedAt: t.Model.CreatedAt,
			UpdatedAt: t.Model.UpdatedAt,
			DeletedAt: t.Model.DeletedAt,
		},
		BusinessID:             t.BusinessID,
		Reference:              t.Reference,
		Amount:                 t.Amount,
		Currency:               t.Currency,
		Narration:              t.Narration,
		RecipientBankCode:      t.RecipientBankCode,
		RecipientAccountNumber: t.RecipientAccountNumber,
		RecipientAccountName:   t.RecipientAccountName,
		Provider:               t.Provider,
		Status:                 t.Status,
		TransactionID:          t.TransactionID,
		ProviderStatus:         t.ProviderStatus,
		ProviderMessage:        t.ProviderMessage,
		Fee:                    t.Fee,
		ProcessedAt:            t.ProcessedAt,
		ProcessingError:        t.ProcessingError,
	}
}

// transferRepository implements repository.TransferRepository
type transferRepository struct {
	db *gorm.DB
}

// NewTransferRepository creates a new transfer repository
func NewTransferRepository(db *gorm.DB) repository.TransferRepository {
	return &transferRepository{db: db}
}

func (r *transferRepository) Create(ctx context.Context, transfer *domain.Transfer) error {
	model := TransferModelFromDomain(transfer)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	*transfer = *model.ToDomain()
	return nil
}

func (r *transferRepository) FindByID(ctx context.Context, id uint) (*domain.Transfer, error) {
	var model TransferModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return model.ToDomain(), nil
}

func (r *transferRepository) FindByReference(ctx context.Context, reference string) (*domain.Transfer, error) {
	var model TransferModel
	if err := r.db.WithContext(ctx).Where("reference = ?", reference).First(&model).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return model.ToDomain(), nil
}

func (r *transferRepository) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.Transfer, int, error) {
	var models []TransferModel
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&TransferModel{}).Where("business_id = ?", businessID).Count(&total).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	// Get paginated results
	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).Where("business_id = ?", businessID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	transfers := make([]*domain.Transfer, len(models))
	for i, model := range models {
		transfers[i] = model.ToDomain()
	}

	return transfers, int(total), nil
}

func (r *transferRepository) Update(ctx context.Context, transfer *domain.Transfer) error {
	model := TransferModelFromDomain(transfer)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	*transfer = *model.ToDomain()
	return nil
}

func (r *transferRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&TransferModel{}, id).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	return nil
}

func (r *transferRepository) WithTx(tx *gorm.DB) repository.TransferRepository {
	return &transferRepository{db: tx}
}
