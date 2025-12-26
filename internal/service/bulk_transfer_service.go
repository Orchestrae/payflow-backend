package service

import (
	"context"
	"crypto/sha512"
	"fmt"
	"log/slog"
	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/internal/service/provider"
	"time"

	"gorm.io/gorm"
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
func (s *bulkTransferService) ExecuteSingleTransfer(ctx context.Context, businessID uint, req *domain.BulkTransferRequest) (*domain.BulkTransferResponse, error) {
	startTime := time.Now()

	slog.Info("Executing single transfer",
		"business_id", businessID,
		"from_account", req.FromAccountNumber,
		"to_account", req.ToAccountNumber,
		"amount", req.Amount,
		"reference", req.Reference,
	)

	// Step 1: Get transfer flow data (account enquiries, prepare transfer request)
	flowData, err := s.GetTransferFlowData(ctx, businessID, req)
	if err != nil {
		processingTime := time.Since(startTime)
		return &domain.BulkTransferResponse{
			Success:        false,
			Reference:      req.Reference,
			Error:          fmt.Sprintf("Failed to prepare transfer data: %v", err),
			ProcessingTime: processingTime,
		}, nil
	}

	// Step 2: Execute the transfer
	result, err := s.executeTransferFlow(ctx, flowData)
	if err != nil {
		processingTime := time.Since(startTime)
		return &domain.BulkTransferResponse{
			Success:        false,
			Reference:      req.Reference,
			Error:          fmt.Sprintf("Failed to execute transfer: %v", err),
			ProcessingTime: processingTime,
		}, nil
	}

	processingTime := time.Since(startTime)

	response := &domain.BulkTransferResponse{
		Success:            result.Success,
		TransferID:         result.TransferID,
		Reference:          req.Reference,
		VFDResponse:        result.VFDResponse,
		ProcessingTime:     processingTime,
		FromAccountDetails: flowData.FromAccountDetails,
		ToAccountDetails:   flowData.ToAccountDetails,
	}

	if !result.Success && result.Error != nil {
		response.Error = result.Error.Error()
	}

	slog.Info("Single transfer completed",
		"success", result.Success,
		"transfer_id", result.TransferID,
		"processing_time", processingTime,
	)

	return response, nil
}

// ExecuteBatchTransfer executes multiple transfers in a batch
func (s *bulkTransferService) ExecuteBatchTransfer(ctx context.Context, businessID uint, req *domain.BulkTransferBatchRequest) (*domain.BulkTransferBatchResponse, error) {
	startTime := time.Now()

	slog.Info("Executing batch transfer",
		"business_id", businessID,
		"total_transfers", len(req.Transfers),
	)

	responses := make([]domain.BulkTransferResponse, len(req.Transfers))
	successfulCount := 0
	failedCount := 0

	// Execute transfers sequentially (can be made concurrent if needed)
	for i, transferReq := range req.Transfers {
		response, err := s.ExecuteSingleTransfer(ctx, businessID, &transferReq)
		if err != nil {
			// If there's an error in the service itself, mark as failed
			responses[i] = domain.BulkTransferResponse{
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

	result := &domain.BulkTransferBatchResponse{
		TotalTransfers:      len(req.Transfers),
		SuccessfulTransfers: successfulCount,
		FailedTransfers:     failedCount,
		Transfers:           responses,
		ProcessingTime:      processingTime,
	}

	slog.Info("Batch transfer completed",
		"total", len(req.Transfers),
		"successful", successfulCount,
		"failed", failedCount,
		"processing_time", processingTime,
	)

	return result, nil
}

// GetTransferFlowData prepares all the data needed for a transfer without executing it
func (s *bulkTransferService) GetTransferFlowData(ctx context.Context, businessID uint, req *domain.BulkTransferRequest) (*domain.TransferFlowData, error) {
	var fromAccountDetails *domain.AccountEnquiryData
	var toAccountDetails *domain.BeneficiaryEnquiryData

	// Step 1: Get from account details (if not provided)
	if req.FromAccountDetails == nil {
		fromResponse, err := s.providerManager.AccountEnquiry(ctx, req.FromAccountNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get from account details: %w", err)
		}
		if fromResponse.Data == nil {
			return nil, fmt.Errorf("no account details found for from account: %s", req.FromAccountNumber)
		}
		fromAccountDetails = fromResponse.Data
	} else {
		fromAccountDetails = req.FromAccountDetails
	}

	// Step 2: Get to account details (if not provided)
	if req.ToAccountDetails == nil {
		toResponse, err := s.providerManager.BeneficiaryEnquiry(ctx, req.ToAccountNumber, req.ToBankCode, req.TransferType)
		if err != nil {
			return nil, fmt.Errorf("failed to get to account details: %w", err)
		}
		if toResponse.Data == nil {
			return nil, fmt.Errorf("no account details found for to account: %s", req.ToAccountNumber)
		}
		toAccountDetails = toResponse.Data
	} else {
		toAccountDetails = req.ToAccountDetails
	}

	// Step 3: Prepare transfer request
	transferRequest := &domain.TransferRequest{
		FromAccount:           fromAccountDetails.AccountNo,
		UniqueSenderAccountId: "", // Optional field
		FromClientId:          fromAccountDetails.ClientId,
		FromClient:            fromAccountDetails.Client,
		FromSavingsId:         fromAccountDetails.AccountId,
		FromBvn:               "", // Will be populated if available
		ToClientId:            toAccountDetails.ClientId,
		ToClient:              toAccountDetails.Name,
		ToSavingsId:           toAccountDetails.Account.ID,
		ToSession:             "", // For inter-bank transfers
		ToBvn:                 toAccountDetails.BVN,
		ToAccount:             toAccountDetails.Account.Number,
		ToBank:                req.ToBankCode,
		Signature:             "", // Will be generated automatically
		Amount:                req.Amount,
		Remark:                req.Remark,
		TransferType:          req.TransferType,
		Reference:             req.Reference,
	}

	// Handle optional fields
	if req.BankCode != "" {
		transferRequest.UniqueSenderAccountId = req.BankCode
	}

	return &domain.TransferFlowData{
		FromAccountDetails: fromAccountDetails,
		ToAccountDetails:   toAccountDetails,
		TransferRequest:    transferRequest,
		BusinessID:         businessID,
	}, nil
}

// executeTransferFlow executes the actual transfer using the prepared data
func (s *bulkTransferService) executeTransferFlow(ctx context.Context, flowData *domain.TransferFlowData) (*domain.TransferFlowResult, error) {
	startTime := time.Now()

	// Start transaction for database operations
	tx := s.txer.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			s.txer.Rollback(tx)
			panic(r)
		}
	}()

	// Generate signature for VFD (if needed)
	signature := s.generateSignature(flowData.TransferRequest.FromAccount, flowData.TransferRequest.ToAccount)
	flowData.TransferRequest.Signature = signature

	// Create transfer record in database
	transfer := &domain.TransferRecord{
		BusinessID:    flowData.BusinessID,
		FromAccount:   flowData.TransferRequest.FromAccount,
		FromClientId:  flowData.TransferRequest.FromClientId,
		FromClient:    flowData.TransferRequest.FromClient,
		FromSavingsId: flowData.TransferRequest.FromSavingsId,
		FromBvn:       &flowData.TransferRequest.FromBvn,
		ToClientId:    flowData.TransferRequest.ToClientId,
		ToClient:      flowData.TransferRequest.ToClient,
		ToSavingsId:   flowData.TransferRequest.ToSavingsId,
		ToSession:     &flowData.TransferRequest.ToSession,
		ToBvn:         &flowData.TransferRequest.ToBvn,
		ToAccount:     flowData.TransferRequest.ToAccount,
		ToBank:        flowData.TransferRequest.ToBank,
		Amount:        flowData.TransferRequest.Amount,
		Remark:        flowData.TransferRequest.Remark,
		TransferType:  flowData.TransferRequest.TransferType,
		Reference:     flowData.TransferRequest.Reference,
		Status:        string(domain.TransferStatusPending),
	}

	// Save transfer record
	var transferRepoTx repository.VFDTransferRepository
	if gormTx, ok := tx.(*gorm.DB); ok {
		transferRepoTx = s.transferRepo.WithTx(gormTx)
		if err := transferRepoTx.Create(ctx, transfer); err != nil {
			s.txer.Rollback(tx)
			processingTime := time.Since(startTime)
			return &domain.TransferFlowResult{
				Success:        false,
				Error:          fmt.Errorf("failed to save transfer record: %w", err),
				ProcessingTime: processingTime,
			}, nil
		}
	} else {
		s.txer.Rollback(tx)
		processingTime := time.Since(startTime)
		return &domain.TransferFlowResult{
			Success:        false,
			Error:          fmt.Errorf("invalid transaction type"),
			ProcessingTime: processingTime,
		}, nil
	}

	// Execute the transfer using the provider manager
	transferResponse, err := s.providerManager.InitiateTransfer(ctx, flowData.BusinessID, flowData.TransferRequest)
	if err != nil {
		// Update transfer record with error
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
		processingTime := time.Since(startTime)
		return &domain.TransferFlowResult{
			Success:        false,
			Error:          err,
			ProcessingTime: processingTime,
		}, nil
	}

	// Update transfer record with success
	transfer.Status = string(domain.TransferStatusSuccess)
	transfer.VFDStatus = transferResponse.Status
	transfer.VFDMessage = transferResponse.Message
	if transferResponse.Data != nil {
		transfer.TxnId = &transferResponse.Data.TxnId
		if transferResponse.Data.SessionId != "" {
			transfer.SessionId = &transferResponse.Data.SessionId
		}
	}
	now := time.Now()
	transfer.ProcessedAt = &now

	if err := transferRepoTx.Update(ctx, transfer); err != nil {
		s.txer.Rollback(tx)
		processingTime := time.Since(startTime)
		return &domain.TransferFlowResult{
			Success:        false,
			Error:          fmt.Errorf("failed to update transfer record: %w", err),
			ProcessingTime: processingTime,
		}, nil
	}

	if err := s.txer.Commit(tx); err != nil {
		s.txer.Rollback(tx)
		processingTime := time.Since(startTime)
		return &domain.TransferFlowResult{
			Success:        false,
			Error:          fmt.Errorf("failed to commit transaction: %w", err),
			ProcessingTime: processingTime,
		}, nil
	}

	processingTime := time.Since(startTime)

	slog.Info("Transfer initiated successfully",
		"transfer_id", transfer.ID,
		"reference", flowData.TransferRequest.Reference,
		"status", transferResponse.Status,
	)

	return &domain.TransferFlowResult{
		Success:        true,
		TransferID:     transfer.ID,
		VFDResponse:    transferResponse,
		ProcessingTime: processingTime,
	}, nil
}

// generateSignature generates a signature using SHA512(fromAccount + toAccount)
func (s *bulkTransferService) generateSignature(fromAccount, toAccount string) string {
	data := fromAccount + toAccount
	hash := sha512.Sum512([]byte(data))
	return fmt.Sprintf("%x", hash)
}
