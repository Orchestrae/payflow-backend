package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"payflow/internal/repository"
)

// ReconciliationService runs periodic balance checks.
type ReconciliationService interface {
	RunDailyReconciliation(ctx context.Context) error
}

type reconciliationService struct {
	walletRepo repository.WalletRepository
	ledgerSvc  LedgerService
}

// NewReconciliationService creates a new reconciliation service.
func NewReconciliationService(walletRepo repository.WalletRepository, ledgerSvc LedgerService) ReconciliationService {
	return &reconciliationService{
		walletRepo: walletRepo,
		ledgerSvc:  ledgerSvc,
	}
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
