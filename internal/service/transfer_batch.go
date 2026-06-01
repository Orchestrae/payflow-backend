// internal/service/transfer_batch.go
// Batch transfer operations extracted from transfer_service.go
package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"payflow/internal/domain"
)

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

	// Create transfer records in database (pending state) using batch insert
	transferRecords := make([]*domain.Transfer, len(req.Transfers))
	for i, t := range req.Transfers {
		transferRecords[i] = &domain.Transfer{
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
	}
	if err := s.transferRepo.CreateBatch(ctx, transferRecords); err != nil {
		slog.Error("Failed to batch-create transfer records", "error", err)
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

	// Record withdrawals for successful transfers + unlock failed ones
	if s.walletService != nil {
		for i, r := range responses {
			if r.Success || r.Status == "processing" || r.Status == "pending" {
				// Record withdrawal for this successful payout
				amount, _ := s.parseAmountToKobo(transferRecords[i].Amount)
				ref := fmt.Sprintf("batch-%s-%s", req.BatchReference, r.Reference)
				s.walletService.RecordWithdrawal(ctx, businessID, transferRecords[i].ID, amount, 0, ref, r.TransactionID)
			} else {
				// Unlock balance for failed transfer
				amount, _ := s.parseAmountToKobo(transferRecords[i].Amount)
				if amount > 0 {
					s.walletService.UnlockBalance(ctx, businessID, amount)
				}
			}
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

