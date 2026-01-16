// Package service contains the bulk transfer service (DEPRECATED)
// This service is deprecated in favor of transfer_service.go which provides
// a cleaner, provider-agnostic implementation.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/internal/service/provider"
)

type bulkTransferService struct {
	providerManager *provider.TransferProviderManager
	transferRepo    repository.VFDTransferRepository
	txer            repository.Transactioner
}

func NewBulkTransferService(
	providerManager *provider.TransferProviderManager,
	transferRepo repository.VFDTransferRepository,
	txer repository.Transactioner,
) BulkTransferService {
	return &bulkTransferService{
		providerManager: providerManager,
		transferRepo:    transferRepo,
		txer:            txer,
	}
}

// ExecuteSingleTransfer executes a complete transfer flow for a single transfer
// DEPRECATED: Use TransferService.ExecuteTransfer instead
func (s *bulkTransferService) ExecuteSingleTransfer(ctx context.Context, businessID uint, req *domain.LegacyBulkTransferRequest) (*domain.LegacyBulkTransferResponse, error) {
	startTime := time.Now()

	slog.Info("Executing single transfer (legacy)",
		"business_id", businessID,
		"to_account", req.ToAccountNumber,
		"amount", req.Amount,
		"reference", req.Reference,
	)

	// Convert to new unified request format
	transferReq := &domain.SingleTransferRequest{
		Reference:     req.Reference,
		Amount:        req.Amount,
		BankCode:      req.ToBankCode,
		AccountNumber: req.ToAccountNumber,
		AccountName:   "", // Will be looked up by provider if needed
		Narration:     req.Remark,
		Currency:      "NGN",
	}

	// If we have beneficiary details, use them
	if req.ToAccountDetails != nil {
		transferReq.AccountName = req.ToAccountDetails.Name
	}

	// Start transaction
	tx := s.txer.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			s.txer.Rollback(tx)
			panic(r)
		}
	}()

	// Create transfer record in database (legacy format)
	transfer := &domain.TransferRecord{
		BusinessID:   businessID,
		ToAccount:    req.ToAccountNumber,
		ToBank:       req.ToBankCode,
		Amount:       req.Amount,
		Remark:       req.Remark,
		TransferType: req.TransferType,
		Reference:    req.Reference,
		Status:       string(domain.TransferStatusPending),
	}

	// Fill in from account details if provided
	if req.FromAccountDetails != nil {
		transfer.FromAccount = req.FromAccountDetails.AccountNo
		transfer.FromClientId = req.FromAccountDetails.ClientId
		transfer.FromClient = req.FromAccountDetails.Client
		transfer.FromSavingsId = req.FromAccountDetails.AccountId
	}

	// Fill in to account details if provided
	if req.ToAccountDetails != nil {
		transfer.ToClient = req.ToAccountDetails.Name
		transfer.ToClientId = req.ToAccountDetails.ClientId
		bvn := req.ToAccountDetails.BVN
		transfer.ToBvn = &bvn
		transfer.ToSavingsId = req.ToAccountDetails.Account.ID
	}

	// Save transfer record
	var transferRepoTx repository.VFDTransferRepository
	if gormTx, ok := tx.(*gorm.DB); ok {
		transferRepoTx = s.transferRepo.WithTx(gormTx)
		if err := transferRepoTx.Create(ctx, transfer); err != nil {
			s.txer.Rollback(tx)
			return &domain.LegacyBulkTransferResponse{
				Success:        false,
				Reference:      req.Reference,
				Error:          fmt.Sprintf("Failed to create transfer record: %v", err),
				ProcessingTime: time.Since(startTime),
			}, nil
		}
	} else {
		s.txer.Rollback(tx)
		return &domain.LegacyBulkTransferResponse{
			Success:        false,
			Reference:      req.Reference,
			Error:          "Invalid transaction type",
			ProcessingTime: time.Since(startTime),
		}, nil
	}

	// Execute transfer via provider manager (new interface)
	result, err := s.providerManager.InitiateTransfer(ctx, transferReq)
	if err != nil {
		transfer.Status = string(domain.TransferStatusFailed)
		transfer.VFDStatus = "99"
		transfer.VFDMessage = "Transfer failed"
		errorMsg := err.Error()
		transfer.ProcessingError = &errorMsg
		now := time.Now()
		transfer.ProcessedAt = &now

		if updateErr := transferRepoTx.Update(ctx, transfer); updateErr != nil {
			slog.Error("Failed to update transfer status to failed", "error", updateErr)
		}
		s.txer.Rollback(tx)

		return &domain.LegacyBulkTransferResponse{
			Success:        false,
			TransferID:     transfer.ID,
			Reference:      req.Reference,
			Error:          err.Error(),
			ProcessingTime: time.Since(startTime),
		}, nil
	}

	// Update transfer record with success
	transfer.Status = string(domain.TransferStatusSuccess)
	transfer.VFDStatus = result.Status
	transfer.VFDMessage = result.Message
	txnID := result.TransactionID
	transfer.TxnId = &txnID
	now := time.Now()
	transfer.ProcessedAt = &now

	if err := transferRepoTx.Update(ctx, transfer); err != nil {
		s.txer.Rollback(tx)
		return &domain.LegacyBulkTransferResponse{
			Success:        false,
			Reference:      req.Reference,
			Error:          fmt.Sprintf("Failed to update transfer record: %v", err),
			ProcessingTime: time.Since(startTime),
		}, nil
	}

	if err := s.txer.Commit(tx); err != nil {
		s.txer.Rollback(tx)
		return &domain.LegacyBulkTransferResponse{
			Success:        false,
			Reference:      req.Reference,
			Error:          fmt.Sprintf("Failed to commit transaction: %v", err),
			ProcessingTime: time.Since(startTime),
		}, nil
	}

	processingTime := time.Since(startTime)

	slog.Info("Transfer completed (legacy)",
		"success", result.Success,
		"transfer_id", transfer.ID,
		"provider", result.Provider,
		"processing_time", processingTime,
	)

	// Build legacy response
	response := &domain.LegacyBulkTransferResponse{
		Success:            result.Success,
		TransferID:         transfer.ID,
		Reference:          req.Reference,
		ProcessingTime:     processingTime,
		FromAccountDetails: req.FromAccountDetails,
		ToAccountDetails:   req.ToAccountDetails,
	}

	// Map to legacy VFD response format for backward compatibility
	response.VFDResponse = &domain.TransferResponse{
		Status:  result.Status,
		Message: result.Message,
		Data: &domain.TransferData{
			TxnId:     result.TransactionID,
			Reference: result.Reference,
		},
	}

	return response, nil
}

// ExecuteBatchTransfer executes multiple transfers in a batch
// DEPRECATED: Use TransferService.ExecuteBatchTransfer instead
func (s *bulkTransferService) ExecuteBatchTransfer(ctx context.Context, businessID uint, req *domain.LegacyBulkTransferBatchRequest) (*domain.LegacyBulkTransferBatchResponse, error) {
	startTime := time.Now()

	slog.Info("Executing batch transfer (legacy)",
		"business_id", businessID,
		"total_transfers", len(req.Transfers),
	)

	responses := make([]domain.LegacyBulkTransferResponse, len(req.Transfers))
	successfulCount := 0
	failedCount := 0

	for i, transferReq := range req.Transfers {
		response, err := s.ExecuteSingleTransfer(ctx, businessID, &transferReq)
		if err != nil {
			responses[i] = domain.LegacyBulkTransferResponse{
				Success:        false,
				Reference:      transferReq.Reference,
				Error:          fmt.Sprintf("Service error: %v", err),
				ProcessingTime: time.Since(startTime),
			}
			failedCount++
		} else {
			responses[i] = *response
			if response.Success {
				successfulCount++
			} else {
				failedCount++
			}
		}
	}

	processingTime := time.Since(startTime)

	slog.Info("Batch transfer completed (legacy)",
		"total", len(req.Transfers),
		"successful", successfulCount,
		"failed", failedCount,
		"processing_time", processingTime,
	)

	return &domain.LegacyBulkTransferBatchResponse{
		TotalTransfers:      len(req.Transfers),
		SuccessfulTransfers: successfulCount,
		FailedTransfers:     failedCount,
		Transfers:           responses,
		ProcessingTime:      processingTime,
	}, nil
}

// GetTransferFlowData prepares all the data needed for a transfer without executing it
// DEPRECATED: This is VFD-specific and not needed with the new provider-agnostic design
func (s *bulkTransferService) GetTransferFlowData(ctx context.Context, businessID uint, req *domain.LegacyBulkTransferRequest) (*domain.TransferFlowData, error) {
	var fromAccountDetails *domain.AccountEnquiryData
	var toAccountDetails *domain.BeneficiaryEnquiryData

	// Get from account details if not provided
	if req.FromAccountDetails == nil && req.FromAccountNumber != "" {
		fromResponse, err := s.providerManager.AccountEnquiry(ctx, req.FromAccountNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get from account details: %w", err)
		}
		if fromResponse != nil && fromResponse.Data != nil {
			fromAccountDetails = fromResponse.Data
		}
	} else {
		fromAccountDetails = req.FromAccountDetails
	}

	// Get to account details if not provided
	if req.ToAccountDetails == nil {
		toResponse, err := s.providerManager.BeneficiaryEnquiry(ctx, req.ToAccountNumber, req.ToBankCode, req.TransferType)
		if err != nil {
			return nil, fmt.Errorf("failed to get to account details: %w", err)
		}
		if toResponse != nil && toResponse.Data != nil {
			toAccountDetails = toResponse.Data
		}
	} else {
		toAccountDetails = req.ToAccountDetails
	}

	// Build transfer request
	transferRequest := &domain.TransferRequest{
		ToAccount:    req.ToAccountNumber,
		ToBank:       req.ToBankCode,
		Amount:       req.Amount,
		Remark:       req.Remark,
		TransferType: req.TransferType,
		Reference:    req.Reference,
	}

	if fromAccountDetails != nil {
		transferRequest.FromAccount = fromAccountDetails.AccountNo
		transferRequest.FromClientId = fromAccountDetails.ClientId
		transferRequest.FromClient = fromAccountDetails.Client
		transferRequest.FromSavingsId = fromAccountDetails.AccountId
	}

	if toAccountDetails != nil {
		transferRequest.ToClient = toAccountDetails.Name
		transferRequest.ToClientId = toAccountDetails.ClientId
		transferRequest.ToSavingsId = toAccountDetails.Account.ID
		transferRequest.ToBvn = toAccountDetails.BVN
	}

	return &domain.TransferFlowData{
		FromAccountDetails: fromAccountDetails,
		ToAccountDetails:   toAccountDetails,
		TransferRequest:    transferRequest,
		BusinessID:         businessID,
	}, nil
}
