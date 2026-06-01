package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/internal/service/provider"
)

// TransferConfig holds configuration for transfer limits
type TransferConfig struct {
	MinAmount int64
	MaxAmount int64
}

// TransferService defines the business logic for transfer operations.
type TransferService interface {
	// ExecuteTransfer executes a single transfer
	ExecuteTransfer(ctx context.Context, businessID uint, req *domain.SingleTransferRequest) (*domain.SingleTransferResponse, error)

	// ExecuteBatchTransfer executes multiple transfers in a batch
	ExecuteBatchTransfer(ctx context.Context, businessID uint, req *domain.BulkTransferRequest) (*domain.BulkTransferResponse, error)

	// GetTransferByID gets a specific transfer record by ID
	GetTransferByID(ctx context.Context, id uint) (*domain.Transfer, error)

	// ListTransfers lists transfer records for a business
	ListTransfers(ctx context.Context, businessID uint, page, limit int) ([]*domain.Transfer, int, error)

	// RetryTransfer retries a failed transfer
	RetryTransfer(ctx context.Context, businessID uint, transferID uint) (*domain.SingleTransferResponse, error)
}

// transferService implements TransferService
type transferService struct {
	providerManager    *provider.TransferProviderManager
	transferRepo       repository.TransferRepository
	userRepo           repository.UserRepository
	walletService      WalletService // Optional: can be nil if not available
	orgProviderKeySvc  OrgProviderSettingsService // Optional: org-level key overrides
	txer               repository.Transactioner
	config             TransferConfig
}

// NewTransferService creates a new transfer service
func NewTransferService(
	providerManager *provider.TransferProviderManager,
	transferRepo repository.TransferRepository,
	userRepo repository.UserRepository,
	txer repository.Transactioner,
	config TransferConfig,
) TransferService {
	return &transferService{
		providerManager: providerManager,
		transferRepo:    transferRepo,
		userRepo:        userRepo,
		walletService:   nil, // Can be set via SetWalletService if needed
		txer:            txer,
		config:          config,
	}
}

// SetWalletService sets the wallet service for balance checking
func (s *transferService) SetWalletService(walletService WalletService) {
	s.walletService = walletService
}

// SetOrgProviderKeySvc sets the org provider key service for org-level key overrides
func (s *transferService) SetOrgProviderKeySvc(svc OrgProviderSettingsService) {
	s.orgProviderKeySvc = svc
}

// ExecuteTransfer executes a single transfer using the provider manager
func (s *transferService) ExecuteTransfer(ctx context.Context, businessID uint, req *domain.SingleTransferRequest) (*domain.SingleTransferResponse, error) {
	startTime := time.Now()

	// Prepare the request with defaults and generated values
	s.prepareTransferRequest(businessID, req)

	slog.Info("Executing transfer",
		"business_id", businessID,
		"reference", req.Reference,
		"amount", req.Amount,
		"bank_code", req.BankCode,
		"account_number", req.AccountNumber,
	)

	// Validate transfer amount against configured limits
	if err := s.validateTransferAmount(req.Amount); err != nil {
		slog.Warn("Transfer amount validation failed",
			"reference", req.Reference,
			"amount", req.Amount,
			"error", err.Error(),
		)
		return &domain.SingleTransferResponse{
			Success:        false,
			Reference:      req.Reference,
			Status:         "failed",
			Error:          err.Error(),
			ProcessingTime: time.Since(startTime),
		}, nil
	}

	// Check balance if wallet service is available
	if s.walletService != nil {
		amountInt64, err := s.parseAmountToKobo(req.Amount)
		if err != nil {
			return &domain.SingleTransferResponse{
				Success:        false,
				Reference:      req.Reference,
				Status:         "failed",
				Error:          fmt.Sprintf("Invalid amount format: %v", err),
				ProcessingTime: time.Since(startTime),
			}, nil
		}

		// Check balance (includes checking locked balance)
		if err := s.walletService.CheckBalance(ctx, businessID, amountInt64); err != nil {
			slog.Warn("Insufficient balance for transfer",
				"reference", req.Reference,
				"amount", req.Amount,
				"error", err.Error(),
			)
			return &domain.SingleTransferResponse{
				Success:        false,
				Reference:      req.Reference,
				Status:         "failed",
				Error:          err.Error(),
				ProcessingTime: time.Since(startTime),
			}, nil
		}
	}

	// Get business user email for provider-specific needs (e.g., Korapay requires customer email)
	s.enrichWithBusinessEmail(ctx, businessID, req)

	// Lock balance if wallet service is available
	amountInt64 := int64(0)
	balanceLocked := false
	withdrawalRecorded := false // Track if withdrawal was recorded (to prevent double unlock)
	if s.walletService != nil {
		var parseErr error
		amountInt64, parseErr = s.parseAmountToKobo(req.Amount)
		if parseErr == nil {
			if lockErr := s.walletService.LockBalance(ctx, businessID, amountInt64); lockErr != nil {
				return &domain.SingleTransferResponse{
					Success:        false,
					Reference:      req.Reference,
					Status:         "failed",
					Error:          fmt.Sprintf("Failed to lock balance: %v", lockErr),
					ProcessingTime: time.Since(startTime),
				}, nil
			}
			balanceLocked = true
		}
	}

	// Unlock balance on failure (deferred until function returns)
	// Note: defer closure captures variables by reference, so withdrawalRecorded will be checked at execute time
	defer func() {
		if balanceLocked && !withdrawalRecorded && s.walletService != nil {
			// Only unlock if withdrawal wasn't recorded (which already handled balance)
			if unlockErr := s.walletService.UnlockBalance(ctx, businessID, amountInt64); unlockErr != nil {
				slog.Error("Failed to unlock balance", "error", unlockErr)
			}
		}
	}()

	// Create transfer record in database (pending state)
	transfer := &domain.Transfer{
		BusinessID:             businessID,
		Reference:              req.Reference,
		Amount:                 req.Amount,
		Currency:               req.Currency,
		Narration:              req.Narration,
		RecipientBankCode:      req.BankCode,
		RecipientAccountNumber: req.AccountNumber,
		RecipientAccountName:   req.AccountName,
		Status:                 "pending",
	}

	if err := s.transferRepo.Create(ctx, transfer); err != nil {
		return &domain.SingleTransferResponse{
			Success:        false,
			Reference:      req.Reference,
			Status:         "failed",
			Error:          fmt.Sprintf("Failed to create transfer record: %v", err),
			ProcessingTime: time.Since(startTime),
		}, nil
	}

	// Execute transfer: use preferred provider if specified, otherwise default+fallback chain
	var result *domain.TransferResult
	var err error
	if req.PreferredProvider != "" {
		specificProvider, exists := s.providerManager.GetProvider(req.PreferredProvider)
		if !exists {
			transfer.Status = "failed"
			transfer.ProcessingError = strPtr(fmt.Sprintf("provider '%s' not available", req.PreferredProvider))
			s.transferRepo.Update(ctx, transfer)
			return &domain.SingleTransferResponse{
				Success:        false,
				Reference:      req.Reference,
				Status:         "failed",
				Error:          fmt.Sprintf("provider '%s' not available", req.PreferredProvider),
				ProcessingTime: time.Since(startTime),
			}, nil
		}
		result, err = specificProvider.InitiateTransfer(ctx, req)
	} else {
		result, err = s.providerManager.InitiateTransfer(ctx, req)
	}
	if err != nil {
		// Update transfer record with failure
		transfer.Status = "failed"
		transfer.ProcessingError = strPtr(err.Error())
		now := time.Now()
		transfer.ProcessedAt = &now

		if updateErr := s.transferRepo.Update(ctx, transfer); updateErr != nil {
			slog.Error("Failed to update transfer status to failed", "error", updateErr)
		}

		return &domain.SingleTransferResponse{
			Success:        false,
			TransferID:     transfer.ID,
			Reference:      req.Reference,
			Status:         "failed",
			Error:          err.Error(),
			ProcessingTime: time.Since(startTime),
		}, nil
	}

	// Update transfer record with success
	transfer.Provider = string(result.Provider)
	transfer.Status = result.Status
	transfer.TransactionID = result.TransactionID
	transfer.ProviderStatus = result.Status
	transfer.ProviderMessage = result.Message
	transfer.Fee = result.Fee
	now := time.Now()
	transfer.ProcessedAt = &now

	if err := s.transferRepo.Update(ctx, transfer); err != nil {
		slog.Error("Failed to update transfer record", "error", err)
		// Don't fail the response - transfer was successful
	}

	// Record withdrawal in wallet if service is available and transfer was successful
	if s.walletService != nil && result.Success && result.Status == "success" {
		feeInt64 := int64(0)
		if result.Fee != "" {
			if parsed, parseErr := s.parseAmountToKobo(result.Fee); parseErr == nil {
				feeInt64 = parsed
			}
		}
		if withdrawErr := s.walletService.RecordWithdrawal(ctx, businessID, transfer.ID, amountInt64, feeInt64, req.Reference, result.TransactionID); withdrawErr != nil {
			slog.Error("Failed to record withdrawal in wallet", "error", withdrawErr)
			// Don't fail the response - transfer was successful, just wallet recording failed
		} else {
			withdrawalRecorded = true // Set flag so defer won't unlock
		}
	}

	processingTime := time.Since(startTime)

	slog.Info("Transfer completed",
		"success", result.Success,
		"transfer_id", transfer.ID,
		"provider", result.Provider,
		"status", result.Status,
		"processing_time", processingTime,
	)

	return &domain.SingleTransferResponse{
		Success:        result.Success,
		TransferID:     transfer.ID,
		Reference:      req.Reference,
		TransactionID:  result.TransactionID,
		Status:         result.Status,
		Message:        result.Message,
		Provider:       result.Provider,
		Fee:            result.Fee,
		ProcessingTime: processingTime,
	}, nil
}


// GetTransferByID gets a specific transfer record by ID
func (s *transferService) GetTransferByID(ctx context.Context, id uint) (*domain.Transfer, error) {
	return s.transferRepo.FindByID(ctx, id)
}

// ListTransfers lists transfer records for a business
func (s *transferService) ListTransfers(ctx context.Context, businessID uint, page, limit int) ([]*domain.Transfer, int, error) {
	return s.transferRepo.FindByBusinessID(ctx, businessID, page, limit)
}

// RetryTransfer retries a failed transfer by re-executing it.
func (s *transferService) RetryTransfer(ctx context.Context, businessID uint, transferID uint) (*domain.SingleTransferResponse, error) {
	transfer, err := s.transferRepo.FindByID(ctx, transferID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	// Verify business ownership
	if transfer.BusinessID != businessID {
		return nil, domain.ErrForbidden
	}

	// Only failed transfers can be retried
	if transfer.Status != "failed" {
		return nil, fmt.Errorf("%w: only failed transfers can be retried (current status: %s)", domain.ErrValidationFailed, transfer.Status)
	}

	// Build a new transfer request from the existing record
	req := &domain.SingleTransferRequest{
		Reference:     transfer.Reference + "-retry",
		Amount:        transfer.Amount,
		BankCode:      transfer.RecipientBankCode,
		AccountNumber: transfer.RecipientAccountNumber,
		AccountName:   transfer.RecipientAccountName,
		Narration:     transfer.Narration,
		Currency:      transfer.Currency,
	}

	// Re-execute
	return s.ExecuteTransfer(ctx, businessID, req)
}

// prepareTransferRequest fills in defaults and generates values for missing fields
func (s *transferService) prepareTransferRequest(businessID uint, req *domain.SingleTransferRequest) {
	// Generate reference if not provided
	if req.Reference == "" {
		req.Reference = s.generateReference(businessID)
	}

	// Set default currency
	if req.Currency == "" {
		req.Currency = "NGN"
	}

	// Set default narration
	if req.Narration == "" {
		req.Narration = "Transfer"
	}

	// Set business ID
	req.BusinessID = businessID
}

// enrichWithBusinessEmail gets the business admin's email for provider-specific needs
func (s *transferService) enrichWithBusinessEmail(ctx context.Context, businessID uint, req *domain.SingleTransferRequest) {
	// Try to get business admin email for provider-specific requirements
	user, err := s.userRepo.FindBusinessAdmin(ctx, businessID)
	if err != nil {
		slog.Debug("Could not get business admin email, will use fallback", "error", err)
		// Use a fallback email pattern
		req.BusinessEmail = fmt.Sprintf("business-%d@payflow.local", businessID)
		return
	}
	req.BusinessEmail = user.Email
}

// generateReference generates a unique reference for a transfer
// Format: TRF-{businessID}-{timestamp}-{random}
func (s *transferService) generateReference(businessID uint) string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomPart := fmt.Sprintf("%x", randomBytes)
	return fmt.Sprintf("TRF-%d-%d-%s", businessID, timestamp, randomPart)
}

// validateTransferAmount validates the transfer amount (in NGN) against configured limits (also in NGN).
func (s *transferService) validateTransferAmount(amountStr string) error {
	// Parse amount in NGN (main currency unit). May contain decimals like "999.99".
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		// Try parsing as float in case it has decimals (e.g., "999.99")
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return fmt.Errorf("invalid amount format: %s", amountStr)
		}
		amount = int64(math.Round(amountFloat))
	}

	// Check minimum amount (config limits are in NGN)
	if s.config.MinAmount > 0 && amount < s.config.MinAmount {
		return domain.NewTransferAmountBelowMinError(amountStr, s.config.MinAmount)
	}

	// Check maximum amount (config limits are in NGN)
	if s.config.MaxAmount > 0 && amount > s.config.MaxAmount {
		return domain.NewTransferAmountAboveMaxError(amountStr, s.config.MaxAmount)
	}

	return nil
}

// parseAmountToKobo parses an amount string in NGN (e.g., "1000" or "999.99") to int64 kobo.
// Input is in main currency units (NGN) and output is in smallest unit (kobo).
// For example: "999.99" -> 99999 kobo, "1000" -> 100000 kobo.
func (s *transferService) parseAmountToKobo(amountStr string) (int64, error) {
	// Always parse as float to handle both "1000" and "999.99" uniformly
	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format: %s", amountStr)
	}

	// Convert to kobo (multiply by 100) and round to avoid float truncation.
	// math.Round ensures "999.99" * 100 = 99999.0 instead of truncating to 99998.
	amountInKobo := int64(math.Round(amountFloat * 100))
	return amountInKobo, nil
}

// Helper function
func strPtr(s string) *string {
	return &s
}
