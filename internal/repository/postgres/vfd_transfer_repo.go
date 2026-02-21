package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"
	"time"

	"gorm.io/gorm"
)

type TransferRecord struct {
	gorm.Model
	BusinessID      uint
	FromAccount     string  `gorm:"size:20;index"`
	FromClientId    string  `gorm:"size:20"`
	FromClient      string  `gorm:"size:255"`
	FromSavingsId   string  `gorm:"size:20"`
	FromBvn         *string `gorm:"size:11"`
	ToClientId      string  `gorm:"size:20"`
	ToClient        string  `gorm:"size:255"`
	ToSavingsId     string  `gorm:"size:20"`
	ToSession       *string `gorm:"size:50"`
	ToBvn           *string `gorm:"size:11"`
	ToAccount       string  `gorm:"size:20;index"`
	ToBank          string  `gorm:"size:10"`
	Amount          string  `gorm:"size:20"`
	Remark          string  `gorm:"size:500"`
	TransferType    string  `gorm:"size:10"`
	Reference       string  `gorm:"size:100;uniqueIndex"`
	TxnId           *string `gorm:"size:100"`
	SessionId       *string `gorm:"size:50"`
	Status          string  `gorm:"size:20;default:'pending'"`
	VFDStatus       string  `gorm:"size:10"`
	VFDMessage      string  `gorm:"size:255"`
	ProcessedAt     *time.Time
	ProcessingError *string `gorm:"size:1000"`
}

type vfdTransferRepository struct {
	db *gorm.DB
}

func NewVFDTransferRepository(db *gorm.DB) repository.VFDTransferRepository {
	return &vfdTransferRepository{db: db}
}

func (r *vfdTransferRepository) Create(ctx context.Context, transfer *domain.TransferRecord) error {
	model := TransferRecordFromDomain(transfer)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	*transfer = *model.ToDomain()
	return nil
}

func (r *vfdTransferRepository) FindByID(ctx context.Context, id uint) (*domain.TransferRecord, error) {
	var model TransferRecord
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return model.ToDomain(), nil
}

func (r *vfdTransferRepository) FindByReference(ctx context.Context, reference string) (*domain.TransferRecord, error) {
	var model TransferRecord
	if err := r.db.WithContext(ctx).Where("reference = ?", reference).First(&model).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return model.ToDomain(), nil
}

func (r *vfdTransferRepository) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.TransferRecord, int, error) {
	var models []TransferRecord
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&TransferRecord{}).Where("business_id = ?", businessID).Count(&total).Error; err != nil {
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

	transfers := make([]*domain.TransferRecord, len(models))
	for i, model := range models {
		transfers[i] = model.ToDomain()
	}

	return transfers, int(total), nil
}

func (r *vfdTransferRepository) FindByFromAccount(ctx context.Context, fromAccount string, page, limit int) ([]*domain.TransferRecord, int, error) {
	var models []TransferRecord
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&TransferRecord{}).Where("from_account = ?", fromAccount).Count(&total).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	// Get paginated results
	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).Where("from_account = ?", fromAccount).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	transfers := make([]*domain.TransferRecord, len(models))
	for i, model := range models {
		transfers[i] = model.ToDomain()
	}

	return transfers, int(total), nil
}

func (r *vfdTransferRepository) FindByToAccount(ctx context.Context, toAccount string, page, limit int) ([]*domain.TransferRecord, int, error) {
	var models []TransferRecord
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&TransferRecord{}).Where("to_account = ?", toAccount).Count(&total).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	// Get paginated results
	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).Where("to_account = ?", toAccount).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	transfers := make([]*domain.TransferRecord, len(models))
	for i, model := range models {
		transfers[i] = model.ToDomain()
	}

	return transfers, int(total), nil
}

func (r *vfdTransferRepository) Update(ctx context.Context, transfer *domain.TransferRecord) error {
	model := TransferRecordFromDomain(transfer)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	*transfer = *model.ToDomain()
	return nil
}

func (r *vfdTransferRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&TransferRecord{}, id).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	return nil
}

func (r *vfdTransferRepository) WithTx(tx *gorm.DB) repository.VFDTransferRepository {
	return &vfdTransferRepository{db: tx}
}

// Conversion methods
func (t *TransferRecord) ToDomain() *domain.TransferRecord {
	transfer := &domain.TransferRecord{
		Model: domain.Model{
			ID:        t.Model.ID,
			CreatedAt: t.Model.CreatedAt,
			UpdatedAt: t.Model.UpdatedAt,
			DeletedAt: t.Model.DeletedAt,
		},
		BusinessID:      t.BusinessID,
		FromAccount:     t.FromAccount,
		FromClientId:    t.FromClientId,
		FromClient:      t.FromClient,
		FromSavingsId:   t.FromSavingsId,
		FromBvn:         t.FromBvn,
		ToClientId:      t.ToClientId,
		ToClient:        t.ToClient,
		ToSavingsId:     t.ToSavingsId,
		ToSession:       t.ToSession,
		ToBvn:           t.ToBvn,
		ToAccount:       t.ToAccount,
		ToBank:          t.ToBank,
		Amount:          t.Amount,
		Remark:          t.Remark,
		TransferType:    t.TransferType,
		Reference:       t.Reference,
		TxnId:           t.TxnId,
		SessionId:       t.SessionId,
		Status:          t.Status,
		VFDStatus:       t.VFDStatus,
		VFDMessage:      t.VFDMessage,
		ProcessingError: t.ProcessingError,
	}

	// Handle processed at
	if t.ProcessedAt != nil {
		transfer.ProcessedAt = t.ProcessedAt
	}

	return transfer
}

func TransferRecordFromDomain(t *domain.TransferRecord) *TransferRecord {
	model := &TransferRecord{
		Model: gorm.Model{
			ID:        t.Model.ID,
			CreatedAt: t.Model.CreatedAt,
			UpdatedAt: t.Model.UpdatedAt,
			DeletedAt: t.Model.DeletedAt,
		},
		BusinessID:      t.BusinessID,
		FromAccount:     t.FromAccount,
		FromClientId:    t.FromClientId,
		FromClient:      t.FromClient,
		FromSavingsId:   t.FromSavingsId,
		FromBvn:         t.FromBvn,
		ToClientId:      t.ToClientId,
		ToClient:        t.ToClient,
		ToSavingsId:     t.ToSavingsId,
		ToSession:       t.ToSession,
		ToBvn:           t.ToBvn,
		ToAccount:       t.ToAccount,
		ToBank:          t.ToBank,
		Amount:          t.Amount,
		Remark:          t.Remark,
		TransferType:    t.TransferType,
		Reference:       t.Reference,
		TxnId:           t.TxnId,
		SessionId:       t.SessionId,
		Status:          t.Status,
		VFDStatus:       t.VFDStatus,
		VFDMessage:      t.VFDMessage,
		ProcessingError: t.ProcessingError,
	}

	// Handle processed at
	if t.ProcessedAt != nil {
		model.ProcessedAt = t.ProcessedAt
	}

	return model
}
