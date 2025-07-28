package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"gorm.io/gorm"
)

type VFDWebhookNotification struct {
	gorm.Model
	BusinessID              uint
	Reference               string `gorm:"size:255;index"`
	Amount                  string `gorm:"size:50"`
	AccountNumber           string `gorm:"size:20;index"`
	OriginatorAccountNumber string `gorm:"size:20"`
	OriginatorAccountName   string `gorm:"size:255"`
	OriginatorBank          string `gorm:"size:10"`
	OriginatorNarration     string `gorm:"size:500"`
	Timestamp               gorm.Model
	TransactionChannel      string `gorm:"size:10"`
	SessionID               string `gorm:"size:50;index"`
	InitialCreditRequest    bool   `gorm:"default:false"`
	Status                  string `gorm:"size:20;default:'pending'"`
	ProcessedAt             *gorm.Model
	ProcessingError         *string `gorm:"size:1000"`
}

type vfdWebhookNotificationRepository struct {
	db *gorm.DB
}

func NewVFDWebhookNotificationRepository(db *gorm.DB) repository.VFDWebhookNotificationRepository {
	return &vfdWebhookNotificationRepository{db: db}
}

func (r *vfdWebhookNotificationRepository) Create(ctx context.Context, notification *domain.VFDWebhookNotification) error {
	model := VFDWebhookNotificationFromDomain(notification)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	*notification = *model.ToDomain()
	return nil
}

func (r *vfdWebhookNotificationRepository) FindByID(ctx context.Context, id uint) (*domain.VFDWebhookNotification, error) {
	var model VFDWebhookNotification
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return model.ToDomain(), nil
}

func (r *vfdWebhookNotificationRepository) FindByReference(ctx context.Context, reference string) (*domain.VFDWebhookNotification, error) {
	var model VFDWebhookNotification
	if err := r.db.WithContext(ctx).Where("reference = ?", reference).First(&model).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return model.ToDomain(), nil
}

func (r *vfdWebhookNotificationRepository) FindBySessionID(ctx context.Context, sessionID string) (*domain.VFDWebhookNotification, error) {
	var model VFDWebhookNotification
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&model).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return model.ToDomain(), nil
}

func (r *vfdWebhookNotificationRepository) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.VFDWebhookNotification, int, error) {
	var models []VFDWebhookNotification
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&VFDWebhookNotification{}).Where("business_id = ?", businessID).Count(&total).Error; err != nil {
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

	notifications := make([]*domain.VFDWebhookNotification, len(models))
	for i, model := range models {
		notifications[i] = model.ToDomain()
	}

	return notifications, int(total), nil
}

func (r *vfdWebhookNotificationRepository) FindByAccountNumber(ctx context.Context, accountNumber string, page, limit int) ([]*domain.VFDWebhookNotification, int, error) {
	var models []VFDWebhookNotification
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&VFDWebhookNotification{}).Where("account_number = ?", accountNumber).Count(&total).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	// Get paginated results
	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).Where("account_number = ?", accountNumber).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, 0, DBErrToDomainErr(err)
	}

	notifications := make([]*domain.VFDWebhookNotification, len(models))
	for i, model := range models {
		notifications[i] = model.ToDomain()
	}

	return notifications, int(total), nil
}

func (r *vfdWebhookNotificationRepository) Update(ctx context.Context, notification *domain.VFDWebhookNotification) error {
	model := VFDWebhookNotificationFromDomain(notification)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	*notification = *model.ToDomain()
	return nil
}

func (r *vfdWebhookNotificationRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&VFDWebhookNotification{}, id).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	return nil
}

func (r *vfdWebhookNotificationRepository) WithTx(tx *gorm.DB) repository.VFDWebhookNotificationRepository {
	return &vfdWebhookNotificationRepository{db: tx}
}

// Conversion methods
func (n *VFDWebhookNotification) ToDomain() *domain.VFDWebhookNotification {
	notification := &domain.VFDWebhookNotification{
		Model: domain.Model{
			ID:        n.Model.ID,
			CreatedAt: n.Model.CreatedAt,
			UpdatedAt: n.Model.UpdatedAt,
			DeletedAt: n.Model.DeletedAt,
		},
		BusinessID:              n.BusinessID,
		Reference:               n.Reference,
		Amount:                  n.Amount,
		AccountNumber:           n.AccountNumber,
		OriginatorAccountNumber: n.OriginatorAccountNumber,
		OriginatorAccountName:   n.OriginatorAccountName,
		OriginatorBank:          n.OriginatorBank,
		OriginatorNarration:     n.OriginatorNarration,
		Timestamp:               n.Timestamp.CreatedAt,
		TransactionChannel:      n.TransactionChannel,
		SessionID:               n.SessionID,
		InitialCreditRequest:    n.InitialCreditRequest,
		Status:                  n.Status,
		ProcessingError:         n.ProcessingError,
	}

	// Handle processed at
	if n.ProcessedAt != nil {
		notification.ProcessedAt = &n.ProcessedAt.CreatedAt
	}

	return notification
}

func VFDWebhookNotificationFromDomain(n *domain.VFDWebhookNotification) *VFDWebhookNotification {
	model := &VFDWebhookNotification{
		Model: gorm.Model{
			ID:        n.Model.ID,
			CreatedAt: n.Model.CreatedAt,
			UpdatedAt: n.Model.UpdatedAt,
			DeletedAt: n.Model.DeletedAt,
		},
		BusinessID:              n.BusinessID,
		Reference:               n.Reference,
		Amount:                  n.Amount,
		AccountNumber:           n.AccountNumber,
		OriginatorAccountNumber: n.OriginatorAccountNumber,
		OriginatorAccountName:   n.OriginatorAccountName,
		OriginatorBank:          n.OriginatorBank,
		OriginatorNarration:     n.OriginatorNarration,
		TransactionChannel:      n.TransactionChannel,
		SessionID:               n.SessionID,
		InitialCreditRequest:    n.InitialCreditRequest,
		Status:                  n.Status,
		ProcessingError:         n.ProcessingError,
	}

	// Handle timestamp
	if !n.Timestamp.IsZero() {
		model.Timestamp = gorm.Model{CreatedAt: n.Timestamp}
	}

	// Handle processed at
	if n.ProcessedAt != nil {
		model.ProcessedAt = &gorm.Model{CreatedAt: *n.ProcessedAt}
	}

	return model
}
