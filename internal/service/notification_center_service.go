package service

import (
	"context"

	"github.com/rs/zerolog/log"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// NotificationCenterService manages in-app notifications.
type NotificationCenterService interface {
	Send(ctx context.Context, userID, businessID uint, title, message, notifType, linkURL string)
	SendToBusinessUsers(ctx context.Context, businessID uint, title, message, notifType, linkURL string)
	List(ctx context.Context, userID uint, page, limit int) ([]*domain.Notification, int, error)
	UnreadCount(ctx context.Context, userID uint) (int, error)
	MarkRead(ctx context.Context, id, userID uint) error
	MarkAllRead(ctx context.Context, userID uint) error
}

type notificationCenterService struct {
	notifRepo repository.NotificationRepository
	userRepo  repository.UserRepository
}

// NewNotificationCenterService creates a new notification center service.
func NewNotificationCenterService(notifRepo repository.NotificationRepository, userRepo repository.UserRepository) NotificationCenterService {
	return &notificationCenterService{notifRepo: notifRepo, userRepo: userRepo}
}

// Send creates a notification for a specific user (async).
func (s *notificationCenterService) Send(ctx context.Context, userID, businessID uint, title, message, notifType, linkURL string) {
	go func() {
		n := &domain.Notification{
			UserID:     userID,
			BusinessID: businessID,
			Title:      title,
			Message:    message,
			Type:       notifType,
			LinkURL:    linkURL,
		}
		if err := s.notifRepo.Create(context.Background(), n); err != nil {
			log.Error().Err(err).Uint("user_id", userID).Msg("Failed to create notification")
		}
	}()
}

// SendToBusinessUsers sends a notification to all users in a business (async).
func (s *notificationCenterService) SendToBusinessUsers(ctx context.Context, businessID uint, title, message, notifType, linkURL string) {
	go func() {
		users, err := s.userRepo.FindByBusinessID(context.Background(), businessID)
		if err != nil {
			log.Error().Err(err).Uint("business_id", businessID).Msg("Failed to find users for notification")
			return
		}
		for _, user := range users {
			n := &domain.Notification{
				UserID:     user.ID,
				BusinessID: businessID,
				Title:      title,
				Message:    message,
				Type:       notifType,
				LinkURL:    linkURL,
			}
			if err := s.notifRepo.Create(context.Background(), n); err != nil {
				log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to create notification")
			}
		}
	}()
}

func (s *notificationCenterService) List(ctx context.Context, userID uint, page, limit int) ([]*domain.Notification, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.notifRepo.FindByUserID(ctx, userID, page, limit)
}

func (s *notificationCenterService) UnreadCount(ctx context.Context, userID uint) (int, error) {
	return s.notifRepo.CountUnread(ctx, userID)
}

func (s *notificationCenterService) MarkRead(ctx context.Context, id, userID uint) error {
	return s.notifRepo.MarkAsRead(ctx, id, userID)
}

func (s *notificationCenterService) MarkAllRead(ctx context.Context, userID uint) error {
	return s.notifRepo.MarkAllAsRead(ctx, userID)
}
