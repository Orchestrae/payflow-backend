package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/internal/service/provider"
)

// WalletService defines the business logic for wallet operations.
type WalletService interface {
	// CreateVirtualAccount creates a virtual account for a business
	CreateVirtualAccount(ctx context.Context, businessID uint, req *domain.CreateVirtualAccountRequest) (*domain.VirtualAccountResult, error)

	// GetWallet retrieves wallet details for a business
	GetWallet(ctx context.Context, businessID uint) (*domain.BusinessWallet, error)

	// GetBalance gets the current balance for a business wallet
	GetBalance(ctx context.Context, businessID uint) (int64, error)

	// CheckBalance checks if a business has sufficient balance for a transaction
	CheckBalance(ctx context.Context, businessID uint, amount int64) error

	// LockBalance locks a specified amount for a pending transfer
	LockBalance(ctx context.Context, businessID uint, amount int64) error

	// UnlockBalance unlocks a previously locked amount
	UnlockBalance(ctx context.Context, businessID uint, amount int64) error

	// RecordDeposit records a deposit transaction (called by webhook)
	RecordDeposit(ctx context.Context, businessID uint, notification *domain.DepositNotification) error

	// RecordDepositByAccountReference records a deposit transaction using account reference (for webhooks)
	RecordDepositByAccountReference(ctx context.Context, accountReference string, notification *domain.DepositNotification) error

	// RecordWithdrawal records a withdrawal transaction (linked to a transfer)
	RecordWithdrawal(ctx context.Context, businessID uint, transferID uint, amount int64, fee int64, reference string, providerReference string) error

	// GetTransactions gets transaction history for a business wallet
	GetTransactions(ctx context.Context, businessID uint, page, limit int) ([]*domain.WalletTransaction, int, error)
}

// walletService implements WalletService
type walletService struct {
	walletRepo          repository.WalletRepository
	walletTxRepo        repository.WalletTransactionRepository
	virtualAccountProvider provider.VirtualAccountProvider
	virtualAccountBalancer provider.VirtualAccountBalancer
}

// NewWalletService creates a new wallet service
func NewWalletService(
	walletRepo repository.WalletRepository,
	walletTxRepo repository.WalletTransactionRepository,
	virtualAccountProvider provider.VirtualAccountProvider,
) WalletService {
	// Cast to balancer if supported
	var balancer provider.VirtualAccountBalancer
	if b, ok := virtualAccountProvider.(provider.VirtualAccountBalancer); ok {
		balancer = b
	}

	return &walletService{
		walletRepo:          walletRepo,
		walletTxRepo:        walletTxRepo,
		virtualAccountProvider: virtualAccountProvider,
		virtualAccountBalancer: balancer,
	}
}

// CreateVirtualAccount creates a virtual account for a business
func (s *walletService) CreateVirtualAccount(ctx context.Context, businessID uint, req *domain.CreateVirtualAccountRequest) (*domain.VirtualAccountResult, error) {
	// Check if wallet already exists
	existingWallet, err := s.walletRepo.FindByBusinessID(ctx, businessID)
	if err == nil && existingWallet != nil {
		return nil, fmt.Errorf("wallet already exists for business %d", businessID)
	}

	// Generate account reference if not provided
	if req.AccountReference == "" {
		req.AccountReference = fmt.Sprintf("payflow-va-%d-%d", businessID, time.Now().Unix())
	}

	// Set business ID
	req.BusinessID = businessID

	// Create virtual account via provider
	result, err := s.virtualAccountProvider.CreateVirtualAccount(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual account: %w", err)
	}

	// Save wallet to database
	wallet := &domain.BusinessWallet{
		BusinessID:              businessID,
		Balance:                 0, // Initialize with zero balance
		LockedBalance:           0,
		Currency:                result.Currency,
		BalanceUpdatedAt:        &result.CreatedAt,
		VirtualAccountNumber:    result.AccountNumber,
		VirtualAccountBankCode:  result.BankCode,
		VirtualAccountBankName:  result.BankName,
		VirtualAccountReference: result.AccountReference,
		VirtualAccountUniqueID:  result.UniqueID,
		VirtualAccountStatus:    result.AccountStatus,
		Provider:                result.Provider,
	}

	// Store provider metadata if available (for future extensibility)
	if result.Success {
		metadata := map[string]interface{}{
			"created_at": result.CreatedAt,
		}
		if metadataBytes, err := json.Marshal(metadata); err == nil {
			wallet.ProviderMetadata = string(metadataBytes)
		}
	}

	if err := s.walletRepo.Create(ctx, wallet); err != nil {
		return nil, fmt.Errorf("failed to save wallet to database: %w", err)
	}

	return result, nil
}

// GetWallet retrieves wallet details for a business
func (s *walletService) GetWallet(ctx context.Context, businessID uint) (*domain.BusinessWallet, error) {
	wallet, err := s.walletRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	return wallet, nil
}

// GetBalance gets the current balance for a business wallet
func (s *walletService) GetBalance(ctx context.Context, businessID uint) (int64, error) {
	wallet, err := s.walletRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		return 0, fmt.Errorf("failed to get wallet balance: %w", err)
	}
	return wallet.Balance, nil
}

// CheckBalance checks if a business has sufficient balance for a transaction
func (s *walletService) CheckBalance(ctx context.Context, businessID uint, amount int64) error {
	wallet, err := s.walletRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		return fmt.Errorf("wallet not found: %w", err)
	}

	availableBalance := wallet.Balance - wallet.LockedBalance
	if availableBalance < amount {
		return fmt.Errorf("insufficient balance: available %d, required %d", availableBalance, amount)
	}

	return nil
}

// LockBalance locks a specified amount for a pending transfer
func (s *walletService) LockBalance(ctx context.Context, businessID uint, amount int64) error {
	_, err := s.walletRepo.IncrementLocked(ctx, businessID, amount)
	if err != nil {
		return fmt.Errorf("insufficient balance or failed to lock: %w", err)
	}
	return nil
}

// UnlockBalance unlocks a previously locked amount
func (s *walletService) UnlockBalance(ctx context.Context, businessID uint, amount int64) error {
	_, err := s.walletRepo.DecrementLocked(ctx, businessID, amount)
	if err != nil {
		return fmt.Errorf("failed to unlock balance: %w", err)
	}
	return nil
}

// RecordDeposit records a deposit transaction (called by webhook)
func (s *walletService) RecordDeposit(ctx context.Context, businessID uint, notification *domain.DepositNotification) error {
	// Check if deposit already processed (idempotency check)
	existingTx, err := s.walletTxRepo.FindByReference(ctx, notification.Reference)
	if err == nil && existingTx != nil {
		return nil
	}

	// Get current balance before the atomic update
	wallet, err := s.walletRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		return fmt.Errorf("wallet not found: %w", err)
	}
	balanceBefore := wallet.Balance

	// Atomically increment balance (prevents race conditions on concurrent deposits)
	updatedWallet, err := s.walletRepo.IncrementBalance(ctx, businessID, notification.Amount)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Record transaction
	tx := &domain.WalletTransaction{
		BusinessID:        businessID,
		TransactionType:   domain.WalletTransactionDeposit,
		Amount:            notification.Amount,
		BalanceBefore:     balanceBefore,
		BalanceAfter:      updatedWallet.Balance,
		Currency:          notification.Currency,
		Reference:         notification.Reference,
		ProviderReference: notification.Reference,
		Description:       notification.Description,
		Status:            notification.Status,
		ProcessedAt:       &notification.ProcessedAt,
	}

	if err := s.walletTxRepo.Create(ctx, tx); err != nil {
		// Retry once before rolling back (network glitch, connection pool)
		if retryErr := s.walletTxRepo.Create(ctx, tx); retryErr != nil {
			// Both attempts failed — rollback balance atomically
			if _, rollbackErr := s.walletRepo.IncrementBalance(ctx, businessID, -notification.Amount); rollbackErr != nil {
				// CRITICAL: Both transaction record AND rollback failed
				// Balance is inflated — needs manual reconciliation
				log.Error().Err(rollbackErr).
					Int64("amount", notification.Amount).
					Uint("business_id", businessID).
					Str("reference", notification.Reference).
					Msg("CRITICAL: Deposit rollback failed — balance may be inconsistent, manual reconciliation needed")
			}
			return fmt.Errorf("failed to record deposit transaction: %w", retryErr)
		}
	}

	return nil
}

// RecordWithdrawal records a withdrawal transaction (linked to a transfer)
func (s *walletService) RecordWithdrawal(ctx context.Context, businessID uint, transferID uint, amount int64, fee int64, reference string, providerReference string) error {
	// Idempotency check: if a withdrawal with this reference already exists, skip
	existingTx, err := s.walletTxRepo.FindByReference(ctx, reference)
	if err == nil && existingTx != nil {
		return nil
	}

	// Calculate total debit (amount + fee)
	totalDebit := amount + fee

	// Get current balance for early validation and transaction record
	wallet, err := s.walletRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		return fmt.Errorf("wallet not found: %w", err)
	}
	balanceBefore := wallet.Balance

	if balanceBefore < totalDebit {
		return fmt.Errorf("insufficient balance for withdrawal: balance %d, required %d", balanceBefore, totalDebit)
	}

	// Atomically deduct balance and unlock the locked amount (prevents race conditions)
	updatedWallet, err := s.walletRepo.DecrementBalanceAndLocked(ctx, businessID, totalDebit, amount)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Record withdrawal transaction
	now := time.Now()
	tx := &domain.WalletTransaction{
		BusinessID:        businessID,
		TransactionType:   domain.WalletTransactionWithdrawal,
		Amount:            totalDebit,
		BalanceBefore:     balanceBefore,
		BalanceAfter:      updatedWallet.Balance,
		Currency:          updatedWallet.Currency,
		Reference:         reference,
		ProviderReference: providerReference,
		Description:       fmt.Sprintf("Transfer withdrawal: %s", reference),
		TransferID:        &transferID,
		Status:            "completed",
		ProcessedAt:       &now,
	}

	if err := s.walletTxRepo.Create(ctx, tx); err != nil {
		// Rollback atomically (best effort)
		s.walletRepo.IncrementBalance(ctx, businessID, totalDebit)
		return fmt.Errorf("failed to record withdrawal transaction: %w", err)
	}

	return nil
}

// GetTransactions gets transaction history for a business wallet
func (s *walletService) GetTransactions(ctx context.Context, businessID uint, page, limit int) ([]*domain.WalletTransaction, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	transactions, total, err := s.walletTxRepo.FindByBusinessID(ctx, businessID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, total, nil
}

// RecordDepositByAccountReference records a deposit transaction using account reference (for webhooks)
func (s *walletService) RecordDepositByAccountReference(ctx context.Context, accountReference string, notification *domain.DepositNotification) error {
	// Find wallet by account reference to get businessID
	wallet, err := s.walletRepo.FindByAccountReference(ctx, accountReference)
	if err != nil {
		return fmt.Errorf("wallet not found for account reference %s: %w", accountReference, err)
	}

	// Record deposit using businessID
	return s.RecordDeposit(ctx, wallet.BusinessID, notification)
}
