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
}

// transferService implements TransferService
type transferService struct {
	providerManager *provider.TransferProviderManager
	transferRepo    repository.TransferRepository
	userRepo        repository.UserRepository
	walletService   WalletService // Optional: can be nil if not available
	txer            repository.Transactioner
	config          TransferConfig
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

	// Execute transfer via provider manager (provider selection is from environment config)
	result, err := s.providerManager.InitiateTransfer(ctx, req)
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

// ExecuteBatchTransfer executes multiple transfers efficiently.
// Uses native bulk endpoints when available (Korapay), or concurrent processing (VFD).
func (s *transferService) ExecuteBatchTransfer(ctx context.Context, businessID uint, req *domain.BulkTransferRequest) (*domain.BulkTransferResponse, error) {
	startTime := time.Now()

	slog.Info("Executing batch transfer",
		"business_id", businessID,
		"total_transfers", len(req.Transfers),
	)

	// Prepare the batch request
	s.prepareBulkTransferRequest(businessID, req)

	// Validate all transfer amounts
	for i := range req.Transfers {
		if err := s.validateTransferAmount(req.Transfers[i].Amount); err != nil {
			slog.Warn("Transfer amount validation failed",
				"reference", req.Transfers[i].Reference,
				"amount", req.Transfers[i].Amount,
				"error", err.Error(),
			)
			return &domain.BulkTransferResponse{
				Success:        false,
				BatchReference: req.BatchReference,
				TotalTransfers: len(req.Transfers),
				FailedTransfers: len(req.Transfers),
				Transfers: []domain.SingleTransferResponse{{
					Success: false,
					Reference: req.Transfers[i].Reference,
					Status: "failed",
					Error: err.Error(),
				}},
				ProcessingTime: time.Since(startTime),
			}, nil
		}
	}

	// Create transfer records in database (pending state)
	transferRecords := make([]*domain.Transfer, len(req.Transfers))
	for i, t := range req.Transfers {
		transfer := &domain.Transfer{
			BusinessID:             businessID,
			Reference:              t.Reference,
			Amount:                 t.Amount,
			Currency:               t.Currency,
			Narration:              t.Narration,
			RecipientBankCode:      t.BankCode,
			RecipientAccountNumber: t.AccountNumber,
			RecipientAccountName:   t.AccountName,
			Status:                 "pending",
		}
		if err := s.transferRepo.Create(ctx, transfer); err != nil {
			slog.Error("Failed to create transfer record", "reference", t.Reference, "error", err)
			// Continue with other transfers
		}
		transferRecords[i] = transfer
	}

	// Execute bulk transfer via provider manager (handles native vs concurrent logic)
	result, err := s.providerManager.InitiateBulkTransfer(ctx, req)
	if err != nil {
		slog.Error("Bulk transfer failed", "error", err)
		return &domain.BulkTransferResponse{
			Success:         false,
			BatchReference:  req.BatchReference,
			TotalTransfers:  len(req.Transfers),
			FailedTransfers: len(req.Transfers),
			ProcessingTime:  time.Since(startTime),
		}, nil
	}

	// Update transfer records with results
	responses := s.updateTransferRecordsWithResults(ctx, transferRecords, result)

	processingTime := time.Since(startTime)

	// Count successes/failures
	successCount := 0
	failedCount := 0
	pendingCount := 0
	for _, r := range responses {
		if r.Success {
			successCount++
		} else if r.Status == "pending" || r.Status == "processing" {
			pendingCount++
		} else {
			failedCount++
		}
	}

	slog.Info("Batch transfer completed",
		"total", len(req.Transfers),
		"successful", successCount,
		"failed", failedCount,
		"pending", pendingCount,
		"processing_time", processingTime,
	)

	return &domain.BulkTransferResponse{
		Success:             failedCount == 0,
		BatchReference:      req.BatchReference,
		TotalTransfers:      len(req.Transfers),
		SuccessfulTransfers: successCount,
		FailedTransfers:     failedCount,
		PendingTransfers:    pendingCount,
		Provider:            result.Provider,
		Transfers:           responses,
		ProcessingTime:      processingTime,
	}, nil
}

// prepareBulkTransferRequest fills in defaults and generates values for bulk request
func (s *transferService) prepareBulkTransferRequest(businessID uint, req *domain.BulkTransferRequest) {
	// Generate batch reference if not provided
	if req.BatchReference == "" {
		req.BatchReference = s.generateBatchReference(businessID)
	}

	// Set default currency
	if req.Currency == "" {
		req.Currency = "NGN"
	}

	// Set business context
	req.BusinessID = businessID

	// Get business email
	ctx := context.Background()
	user, err := s.userRepo.FindBusinessAdmin(ctx, businessID)
	if err != nil {
		req.BusinessEmail = fmt.Sprintf("business-%d@payflow.local", businessID)
	} else {
		req.BusinessEmail = user.Email
	}

	// Prepare each transfer
	for i := range req.Transfers {
		s.prepareTransferRequest(businessID, &req.Transfers[i])
	}
}

// generateBatchReference generates a unique reference for a batch
func (s *transferService) generateBatchReference(businessID uint) string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomPart := fmt.Sprintf("%x", randomBytes)
	return fmt.Sprintf("BATCH-%d-%d-%s", businessID, timestamp, randomPart)
}

// updateTransferRecordsWithResults updates database records with provider results
func (s *transferService) updateTransferRecordsWithResults(ctx context.Context, records []*domain.Transfer, result *domain.BulkTransferResult) []domain.SingleTransferResponse {
	responses := make([]domain.SingleTransferResponse, len(records))
	now := time.Now()

	// Create a map of results by reference for quick lookup
	resultMap := make(map[string]*domain.TransferResult)
	for i := range result.TransferResults {
		resultMap[result.TransferResults[i].Reference] = &result.TransferResults[i]
	}

	for i, record := range records {
		if record == nil {
			continue
		}

		// Find matching result
		if tr, ok := resultMap[record.Reference]; ok {
			record.Provider = string(tr.Provider)
			record.Status = tr.Status
			record.TransactionID = tr.TransactionID
			record.ProviderStatus = tr.Status
			record.ProviderMessage = tr.Message
			record.Fee = tr.Fee
			record.ProcessedAt = &now

			if !tr.Success {
				errMsg := tr.Message
				record.ProcessingError = &errMsg
			}

			responses[i] = domain.SingleTransferResponse{
				Success:       tr.Success,
				TransferID:    record.ID,
				Reference:     record.Reference,
				TransactionID: tr.TransactionID,
				Status:        tr.Status,
				Message:       tr.Message,
				Provider:      tr.Provider,
				Fee:           tr.Fee,
			}
		} else {
			// No result found - mark as pending (for native bulk that doesn't return individual results)
			record.Provider = string(result.Provider)
			record.Status = "pending"
			record.ProcessedAt = &now

			responses[i] = domain.SingleTransferResponse{
				Success:    true, // Assume success for native bulk
				TransferID: record.ID,
				Reference:  record.Reference,
				Status:     "pending",
				Provider:   result.Provider,
			}
		}

		// Update record in database
		if err := s.transferRepo.Update(ctx, record); err != nil {
			slog.Error("Failed to update transfer record", "reference", record.Reference, "error", err)
		}
	}

	return responses
}

// GetTransferByID gets a specific transfer record by ID
func (s *transferService) GetTransferByID(ctx context.Context, id uint) (*domain.Transfer, error) {
	return s.transferRepo.FindByID(ctx, id)
}

// ListTransfers lists transfer records for a business
func (s *transferService) ListTransfers(ctx context.Context, businessID uint, page, limit int) ([]*domain.Transfer, int, error) {
	return s.transferRepo.FindByBusinessID(ctx, businessID, page, limit)
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
