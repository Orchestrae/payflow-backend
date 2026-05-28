package service

import (
	"context"

	"github.com/rs/zerolog/log"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// AuditService logs and retrieves audit trail entries.
type AuditService interface {
	Log(ctx context.Context, userID, businessID uint, action, resourceType string, resourceID uint, description, ip string)
	ListByBusiness(ctx context.Context, businessID uint, page, limit int) ([]*domain.AuditLog, int, error)
}

type auditService struct {
	auditRepo repository.AuditRepository
}

// NewAuditService creates a new audit service.
func NewAuditService(auditRepo repository.AuditRepository) AuditService {
	return &auditService{auditRepo: auditRepo}
}

// Log creates an audit entry asynchronously (non-blocking).
func (s *auditService) Log(ctx context.Context, userID, businessID uint, action, resourceType string, resourceID uint, description, ip string) {
	go func() {
		entry := &domain.AuditLog{
			UserID:       userID,
			BusinessID:   businessID,
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Description:  description,
			IPAddress:    ip,
		}
		if err := s.auditRepo.Create(context.Background(), entry); err != nil {
			log.Error().Err(err).Str("action", action).Str("resource", resourceType).Msg("Failed to write audit log")
		}
	}()
}

func (s *auditService) ListByBusiness(ctx context.Context, businessID uint, page, limit int) ([]*domain.AuditLog, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.auditRepo.FindByBusinessID(ctx, businessID, page, limit)
}
