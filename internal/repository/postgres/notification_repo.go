package postgres

import (
	"context"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) repository.NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *notificationRepository) FindByUserID(ctx context.Context, userID uint, page, limit int) ([]*domain.Notification, int, error) {
	var notifications []*domain.Notification
	var total int64

	r.db.WithContext(ctx).Model(&domain.Notification{}).Where("user_id = ?", userID).Count(&total)

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&notifications).Error

	return notifications, int(total), err
}

func (r *notificationRepository) CountUnread(ctx context.Context, userID uint) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Notification{}).
		Where("user_id = ? AND is_read = false", userID).Count(&count).Error
	return int(count), err
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, id, userID uint) error {
	result := r.db.WithContext(ctx).Model(&domain.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return result.Error
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&domain.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}
