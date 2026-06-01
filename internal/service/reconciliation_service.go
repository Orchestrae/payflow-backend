package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog/log"

	"payflow/internal/domain"
	"payflow/internal/platform/paystack"
	"payflow/internal/repository"
)

// ProviderReconciliationResult holds the result of a provider reconciliation check.
type ProviderReconciliationResult struct {
	PaystackBalance int64     `json:"paystack_balance"`
	InternalBalance int64     `json:"internal_balance"`
	Discrepancy     int64     `json:"discrepancy"`
	IsReconciled    bool      `json:"is_reconciled"`
	CheckedAt       time.Time `json:"checked_at"`
}

// ReconciliationService runs periodic balance checks.
type ReconciliationService interface {
	RunDailyReconciliation(ctx context.Context) error
	RunProviderReconciliation(ctx context.Context) (*ProviderReconciliationResult, error)
}

type reconciliationService struct {
	walletRepo      repository.WalletRepository
	ledgerSvc       LedgerService
	paystackClient  *paystack.Client
	notificationSvc NotificationService
	userRepo        repository.UserRepository
	// Discrepancy threshold in kobo (default 100000 = NGN 1,000)
	discrepancyThreshold int64
}

// ReconciliationOption is a functional option for the reconciliation service.
type ReconciliationOption func(*reconciliationService)

// WithPaystackClient sets the Paystack client for provider reconciliation.
func WithPaystackClient(client *paystack.Client) ReconciliationOption {
	return func(s *reconciliationService) {
		s.paystackClient = client
	}
}

// WithNotificationSvc sets the notification service for alert emails.
func WithNotificationSvc(svc NotificationService) ReconciliationOption {
	return func(s *reconciliationService) {
		s.notificationSvc = svc
	}
}

// WithUserRepo sets the user repository for finding super admin emails.
func WithUserRepo(repo repository.UserRepository) ReconciliationOption {
	return func(s *reconciliationService) {
		s.userRepo = repo
	}
}

// WithDiscrepancyThreshold sets a custom discrepancy threshold in kobo.
func WithDiscrepancyThreshold(threshold int64) ReconciliationOption {
	return func(s *reconciliationService) {
		s.discrepancyThreshold = threshold
	}
}

// NewReconciliationService creates a new reconciliation service.
func NewReconciliationService(walletRepo repository.WalletRepository, ledgerSvc LedgerService, opts ...ReconciliationOption) ReconciliationService {
	s := &reconciliationService{
		walletRepo:           walletRepo,
		ledgerSvc:            ledgerSvc,
		discrepancyThreshold: 100000, // default: 100,000 kobo = NGN 1,000
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// RunDailyReconciliation checks all wallets against their ledger balances.
func (s *reconciliationService) RunDailyReconciliation(ctx context.Context) error {
	log.Info().Msg("Starting daily reconciliation run")
	start := time.Now()

	wallets, err := s.walletRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch wallets: %w", err)
	}

	var discrepancies int
	for _, wallet := range wallets {
		result, err := s.ledgerSvc.Reconcile(ctx, wallet.BusinessID, wallet.Balance)
		if err != nil {
			log.Error().Err(err).Uint("business_id", wallet.BusinessID).Msg("Reconciliation failed for business")
			continue
		}

		if !result.IsReconciled {
			discrepancies++
			log.Warn().
				Uint("business_id", wallet.BusinessID).
				Int64("wallet_balance", wallet.Balance).
				Int64("ledger_balance", result.LedgerBalance).
				Int64("discrepancy", result.Discrepancy).
				Msg("RECONCILIATION DISCREPANCY DETECTED")
		}
	}

	duration := time.Since(start)
	log.Info().
		Int("wallets_checked", len(wallets)).
		Int("discrepancies", discrepancies).
		Dur("duration", duration).
		Msg("Daily reconciliation complete")

	return nil
}

// RunProviderReconciliation compares Paystack balance against the sum of all internal wallet balances.
func (s *reconciliationService) RunProviderReconciliation(ctx context.Context) (*ProviderReconciliationResult, error) {
	log.Info().Msg("Starting provider reconciliation run")
	start := time.Now()

	if s.paystackClient == nil {
		return nil, fmt.Errorf("paystack client not configured for provider reconciliation")
	}

	// 1. Fetch Paystack balance
	balanceResp, err := s.paystackClient.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch paystack balance: %w", err)
	}

	var paystackBalance int64
	for _, b := range balanceResp.Data {
		if b.Currency == "NGN" {
			paystackBalance = b.Balance
			break
		}
	}

	// 2. Sum all internal wallet balances
	wallets, err := s.walletRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wallets: %w", err)
	}

	var internalBalance int64
	for _, wallet := range wallets {
		internalBalance += wallet.Balance
	}

	// 3. Calculate discrepancy
	discrepancy := paystackBalance - internalBalance
	absDiscrepancy := int64(math.Abs(float64(discrepancy)))
	isReconciled := absDiscrepancy <= s.discrepancyThreshold

	result := &ProviderReconciliationResult{
		PaystackBalance: paystackBalance,
		InternalBalance: internalBalance,
		Discrepancy:     discrepancy,
		IsReconciled:    isReconciled,
		CheckedAt:       time.Now(),
	}

	duration := time.Since(start)
	logEvent := log.Info().
		Int64("paystack_balance", paystackBalance).
		Int64("internal_balance", internalBalance).
		Int64("discrepancy", discrepancy).
		Bool("is_reconciled", isReconciled).
		Int64("threshold", s.discrepancyThreshold).
		Dur("duration", duration)

	if !isReconciled {
		logEvent.Msg("PROVIDER RECONCILIATION DISCREPANCY DETECTED")
		// Send alert email to super admins
		s.sendDiscrepancyAlert(ctx, result)
	} else {
		logEvent.Msg("Provider reconciliation complete — balances match")
	}

	return result, nil
}

// sendDiscrepancyAlert emails all super admins about a provider reconciliation discrepancy.
func (s *reconciliationService) sendDiscrepancyAlert(ctx context.Context, result *ProviderReconciliationResult) {
	if s.notificationSvc == nil || s.userRepo == nil {
		log.Warn().Msg("Cannot send provider reconciliation alert: notification service or user repo not configured")
		return
	}

	superAdmins, err := s.userRepo.FindByRole(ctx, domain.RoleSuperAdmin)
	if err != nil {
		log.Error().Err(err).Msg("Failed to find super admins for reconciliation alert")
		return
	}

	if len(superAdmins) == 0 {
		log.Warn().Msg("No super admin users found to receive reconciliation alert")
		return
	}

	subject := "PayFlow Alert: Provider Reconciliation Discrepancy"
	body := fmt.Sprintf(
		"A discrepancy has been detected between the Paystack balance and internal wallet balances.\n\n"+
			"Paystack Balance: %d kobo (NGN %.2f)\n"+
			"Internal Balance: %d kobo (NGN %.2f)\n"+
			"Discrepancy: %d kobo (NGN %.2f)\n"+
			"Checked At: %s\n\n"+
			"Please investigate immediately.",
		result.PaystackBalance, float64(result.PaystackBalance)/100,
		result.InternalBalance, float64(result.InternalBalance)/100,
		result.Discrepancy, float64(result.Discrepancy)/100,
		result.CheckedAt.Format(time.RFC3339),
	)

	for _, admin := range superAdmins {
		if err := s.notificationSvc.SendEmail(ctx, admin.Email, subject, body); err != nil {
			log.Error().Err(err).Str("email", admin.Email).Msg("Failed to send reconciliation alert email")
		}
	}
}
