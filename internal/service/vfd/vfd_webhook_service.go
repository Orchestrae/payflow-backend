package vfd

import (
	"context"
	"fmt"
	"log/slog"
	"payflow/internal/domain"
	"payflow/internal/platform/vfd"
	"payflow/internal/repository"
	"payflow/internal/service"
	"time"

	"gorm.io/gorm"
)

type vfdWebhookService struct {
	webhookRepo  repository.VFDWebhookNotificationRepository
	businessRepo repository.BusinessRepository
	vfdService   vfd.VFDService
	txer         repository.Transactioner
}

func NewVFDWebhookService(
	webhookRepo repository.VFDWebhookNotificationRepository,
	businessRepo repository.BusinessRepository,
	vfdService vfd.VFDService,
	txer repository.Transactioner,
) service.VFDWebhookService {
	return &vfdWebhookService{
		webhookRepo:  webhookRepo,
		businessRepo: businessRepo,
		vfdService:   vfdService,
		txer:         txer,
	}
}

func (s *vfdWebhookService) ProcessInwardCreditWebhook(ctx context.Context, notification *domain.VFDWebhookNotification) error {
	slog.Info("Processing inward credit webhook",
		"reference", notification.Reference,
		"amount", notification.Amount,
		"account_number", notification.AccountNumber,
		"session_id", notification.SessionID,
	)

	// Start transaction
	tx := s.txer.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			s.txer.Rollback(tx)
			panic(r)
		}
	}()

	// Find business by account number
	business, err := s.findBusinessByAccountNumber(ctx, notification.AccountNumber)
	if err != nil {
		s.txer.Rollback(tx)
		return fmt.Errorf("failed to find business for account %s: %w", notification.AccountNumber, err)
	}

	// Set business ID
	notification.BusinessID = business.ID
	notification.Status = string(domain.WebhookStatusPending)

	// Save webhook notification
	var webhookRepoTx repository.VFDWebhookNotificationRepository
	if gormTx, ok := tx.(*gorm.DB); ok {
		webhookRepoTx = s.webhookRepo.WithTx(gormTx)
		if err := webhookRepoTx.Create(ctx, notification); err != nil {
			s.txer.Rollback(tx)
			return fmt.Errorf("failed to save webhook notification: %w", err)
		}
	} else {
		s.txer.Rollback(tx)
		return fmt.Errorf("invalid transaction type")
	}

	// Process the webhook (business logic)
	if err := s.processInwardCredit(ctx, notification); err != nil {
		// Mark as failed
		notification.Status = string(domain.WebhookStatusFailed)
		errorMsg := err.Error()
		notification.ProcessingError = &errorMsg
		now := time.Now()
		notification.ProcessedAt = &now

		if updateErr := webhookRepoTx.Update(ctx, notification); updateErr != nil {
			slog.Error("Failed to update webhook status to failed", "error", updateErr)
		}

		s.txer.Rollback(tx)
		return fmt.Errorf("failed to process inward credit: %w", err)
	}

	// Mark as processed
	notification.Status = string(domain.WebhookStatusProcessed)
	now := time.Now()
	notification.ProcessedAt = &now

	if err := webhookRepoTx.Update(ctx, notification); err != nil {
		s.txer.Rollback(tx)
		return fmt.Errorf("failed to update webhook status: %w", err)
	}

	if err := s.txer.Commit(tx); err != nil {
		s.txer.Rollback(tx)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("Successfully processed inward credit webhook",
		"reference", notification.Reference,
		"business_id", business.ID,
	)

	return nil
}

func (s *vfdWebhookService) ProcessInitialInwardCreditWebhook(ctx context.Context, notification *domain.VFDWebhookNotification) error {
	slog.Info("Processing initial inward credit webhook",
		"reference", notification.Reference,
		"amount", notification.Amount,
		"account_number", notification.AccountNumber,
		"session_id", notification.SessionID,
	)

	// Start transaction
	tx := s.txer.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			s.txer.Rollback(tx)
			panic(r)
		}
	}()

	// Find business by account number
	business, err := s.findBusinessByAccountNumber(ctx, notification.AccountNumber)
	if err != nil {
		s.txer.Rollback(tx)
		return fmt.Errorf("failed to find business for account %s: %w", notification.AccountNumber, err)
	}

	// Set business ID and mark as initial credit request
	notification.BusinessID = business.ID
	notification.InitialCreditRequest = true
	notification.Status = string(domain.WebhookStatusPending)

	// Save webhook notification
	var webhookRepoTx repository.VFDWebhookNotificationRepository
	if gormTx, ok := tx.(*gorm.DB); ok {
		webhookRepoTx = s.webhookRepo.WithTx(gormTx)
		if err := webhookRepoTx.Create(ctx, notification); err != nil {
			s.txer.Rollback(tx)
			return fmt.Errorf("failed to save webhook notification: %w", err)
		}
	} else {
		s.txer.Rollback(tx)
		return fmt.Errorf("invalid transaction type")
	}

	// Process the initial credit webhook (business logic)
	if err := s.processInitialInwardCredit(ctx, notification); err != nil {
		// Mark as failed
		notification.Status = string(domain.WebhookStatusFailed)
		errorMsg := err.Error()
		notification.ProcessingError = &errorMsg
		now := time.Now()
		notification.ProcessedAt = &now

		if updateErr := webhookRepoTx.Update(ctx, notification); updateErr != nil {
			slog.Error("Failed to update webhook status to failed", "error", updateErr)
		}

		s.txer.Rollback(tx)
		return fmt.Errorf("failed to process initial inward credit: %w", err)
	}

	// Mark as processed
	notification.Status = string(domain.WebhookStatusProcessed)
	now := time.Now()
	notification.ProcessedAt = &now

	if err := webhookRepoTx.Update(ctx, notification); err != nil {
		s.txer.Rollback(tx)
		return fmt.Errorf("failed to update webhook status: %w", err)
	}

	if err := s.txer.Commit(tx); err != nil {
		s.txer.Rollback(tx)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("Successfully processed initial inward credit webhook",
		"reference", notification.Reference,
		"business_id", business.ID,
	)

	return nil
}

func (s *vfdWebhookService) RetriggerWebhookNotification(ctx context.Context, req *domain.VFDRetriggerRequest) (*domain.VFDRetriggerResponse, error) {
	slog.Info("Retriggering webhook notification",
		"transaction_id", req.TransactionID,
		"session_id", req.SessionID,
		"push_identifier", req.PushIdentifier,
	)

	// Call VFD API to retrigger webhook
	response, err := s.vfdService.RetriggerWebhookNotification(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrigger webhook via VFD API: %w", err)
	}

	slog.Info("Successfully retriggered webhook notification",
		"status", response.Status,
		"message", response.Message,
	)

	return response, nil
}

func (s *vfdWebhookService) ListWebhookNotifications(ctx context.Context, businessID uint, page, limit int) ([]*domain.VFDWebhookNotification, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, total, err := s.webhookRepo.FindByBusinessID(ctx, businessID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list webhook notifications: %w", err)
	}

	return notifications, total, nil
}

func (s *vfdWebhookService) GetWebhookNotificationByID(ctx context.Context, id uint) (*domain.VFDWebhookNotification, error) {
	notification, err := s.webhookRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook notification: %w", err)
	}

	return notification, nil
}

func (s *vfdWebhookService) GetWebhookNotificationsByAccountNumber(ctx context.Context, accountNumber string, page, limit int) ([]*domain.VFDWebhookNotification, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, total, err := s.webhookRepo.FindByAccountNumber(ctx, accountNumber, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get webhook notifications by account number: %w", err)
	}

	return notifications, total, nil
}

// Helper methods

func (s *vfdWebhookService) findBusinessByAccountNumber(ctx context.Context, accountNumber string) (*domain.Business, error) {
	// This is a simplified implementation. In a real scenario, you might have a separate
	// mapping table or the account number might be stored directly in the business table.
	// For now, we'll search through all businesses to find the one with matching VFD account number.

	// Note: This is not efficient for large datasets. In production, you should have
	// a proper index or mapping table for account numbers to business IDs.

	// For now, we'll return an error indicating this needs to be implemented
	// based on your specific business logic and data structure.

	return nil, fmt.Errorf("business lookup by account number not implemented - needs business-specific logic")
}

func (s *vfdWebhookService) processInwardCredit(ctx context.Context, notification *domain.VFDWebhookNotification) error {
	// This is where you implement your business logic for processing inward credits
	// Examples:
	// - Update account balance
	// - Create transaction records
	// - Send notifications to users
	// - Trigger other business processes

	slog.Info("Processing inward credit business logic",
		"amount", notification.Amount,
		"originator", notification.OriginatorAccountName,
		"narration", notification.OriginatorNarration,
	)

	// TODO: Implement your specific business logic here
	// For now, we'll just log the processing

	return nil
}

func (s *vfdWebhookService) processInitialInwardCredit(ctx context.Context, notification *domain.VFDWebhookNotification) error {
	// This is where you implement your business logic for processing initial inward credits
	// This might be different from regular inward credits - perhaps just logging or
	// preliminary processing before the actual settlement

	slog.Info("Processing initial inward credit business logic",
		"amount", notification.Amount,
		"originator", notification.OriginatorAccountName,
		"narration", notification.OriginatorNarration,
	)

	// TODO: Implement your specific business logic here
	// For now, we'll just log the processing

	return nil
}
