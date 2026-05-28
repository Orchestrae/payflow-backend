package postgres

import (
	"context"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) repository.AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *auditRepository) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.AuditLog, int, error) {
	var logs []*domain.AuditLog
	var total int64

	if err := r.db.WithContext(ctx).Model(&domain.AuditLog{}).Where("business_id = ?", businessID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := r.db.WithContext(ctx).Where("business_id = ?", businessID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, int(total), nil
}
