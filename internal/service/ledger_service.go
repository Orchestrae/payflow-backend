package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// ReconciliationResult shows the comparison between ledger and wallet balance.
type ReconciliationResult struct {
	BusinessID     uint  `json:"business_id"`
	WalletBalance  int64 `json:"wallet_balance"`
	LedgerCredits  int64 `json:"ledger_credits"`
	LedgerDebits   int64 `json:"ledger_debits"`
	LedgerBalance  int64 `json:"ledger_balance"`
	Discrepancy    int64 `json:"discrepancy"`
	IsReconciled   bool  `json:"is_reconciled"`
}

// LedgerService provides double-entry accounting operations.
type LedgerService interface {
	RecordDeposit(ctx context.Context, businessID uint, amount int64, reference, description string) error
	RecordWithdrawal(ctx context.Context, businessID uint, amount int64, reference, description string) error
	RecordFee(ctx context.Context, businessID uint, amount int64, reference, description string) error
	GetEntries(ctx context.Context, businessID uint, page, limit int) ([]*domain.LedgerEntry, int, error)
	Reconcile(ctx context.Context, businessID uint, walletBalance int64) (*ReconciliationResult, error)
}

type ledgerService struct {
	ledgerRepo repository.LedgerRepository
}

// NewLedgerService creates a new ledger service.
func NewLedgerService(ledgerRepo repository.LedgerRepository) LedgerService {
	return &ledgerService{ledgerRepo: ledgerRepo}
}

// RecordDeposit creates a double-entry for an incoming deposit.
// Credit wallet (money in), Debit external (source of funds).
func (s *ledgerService) RecordDeposit(ctx context.Context, businessID uint, amount int64, reference, description string) error {
	txnID := generateLedgerTxnID()

	credit := &domain.LedgerEntry{
		BusinessID:  businessID,
		TransactionID: txnID,
		AccountType: domain.AccountWallet,
		EntryType:   domain.EntryCredit,
		Amount:      amount,
		Description: description,
		Reference:   reference,
	}

	debit := &domain.LedgerEntry{
		BusinessID:  businessID,
		TransactionID: txnID,
		AccountType: domain.AccountExternal,
		EntryType:   domain.EntryDebit,
		Amount:      amount,
		Description: description,
		Reference:   reference,
	}

	return s.ledgerRepo.CreatePair(ctx, debit, credit)
}

// RecordWithdrawal creates a double-entry for an outgoing transfer.
// Debit wallet (money out), Credit external (destination).
func (s *ledgerService) RecordWithdrawal(ctx context.Context, businessID uint, amount int64, reference, description string) error {
	txnID := generateLedgerTxnID()

	debit := &domain.LedgerEntry{
		BusinessID:  businessID,
		TransactionID: txnID,
		AccountType: domain.AccountWallet,
		EntryType:   domain.EntryDebit,
		Amount:      amount,
		Description: description,
		Reference:   reference,
	}

	credit := &domain.LedgerEntry{
		BusinessID:  businessID,
		TransactionID: txnID,
		AccountType: domain.AccountExternal,
		EntryType:   domain.EntryCredit,
		Amount:      amount,
		Description: description,
		Reference:   reference,
	}

	return s.ledgerRepo.CreatePair(ctx, debit, credit)
}

// RecordFee creates a double-entry for a platform fee.
// Debit wallet, Credit revenue.
func (s *ledgerService) RecordFee(ctx context.Context, businessID uint, amount int64, reference, description string) error {
	txnID := generateLedgerTxnID()

	debit := &domain.LedgerEntry{
		BusinessID:  businessID,
		TransactionID: txnID,
		AccountType: domain.AccountWallet,
		EntryType:   domain.EntryDebit,
		Amount:      amount,
		Description: description,
		Reference:   reference,
	}

	credit := &domain.LedgerEntry{
		BusinessID:  businessID,
		TransactionID: txnID,
		AccountType: domain.AccountRevenue,
		EntryType:   domain.EntryCredit,
		Amount:      amount,
		Description: description,
		Reference:   reference,
	}

	return s.ledgerRepo.CreatePair(ctx, debit, credit)
}

func (s *ledgerService) GetEntries(ctx context.Context, businessID uint, page, limit int) ([]*domain.LedgerEntry, int, error) {
	if page <= 0 { page = 1 }
	if limit <= 0 { limit = 50 }
	return s.ledgerRepo.FindByBusinessID(ctx, businessID, page, limit)
}

// Reconcile compares ledger balance with wallet balance.
func (s *ledgerService) Reconcile(ctx context.Context, businessID uint, walletBalance int64) (*ReconciliationResult, error) {
	credits, debits, ledgerBalance, err := s.ledgerRepo.Reconcile(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("reconciliation failed: %w", err)
	}

	discrepancy := walletBalance - ledgerBalance

	return &ReconciliationResult{
		BusinessID:    businessID,
		WalletBalance: walletBalance,
		LedgerCredits: credits,
		LedgerDebits:  debits,
		LedgerBalance: ledgerBalance,
		Discrepancy:   discrepancy,
		IsReconciled:  discrepancy == 0,
	}, nil
}

func generateLedgerTxnID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "TXN-" + hex.EncodeToString(b)[:16]
}
