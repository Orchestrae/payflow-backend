package service

import (
	"context"
	"fmt"
	"log/slog"
	"payflow/internal/domain"
	"payflow/internal/platform/vfd"
	"payflow/internal/repository"
	"time"
)

type bulkTransferService struct {
	transferService VFDTransferService
	vfdService      vfd.VFDService
	txer            repository.Transactioner
}

func NewBulkTransferService(
	transferService VFDTransferService,
	vfdService vfd.VFDService,
	txer repository.Transactioner,
) BulkTransferService {
	return &bulkTransferService{
		transferService: transferService,
		vfdService:      vfdService,
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
		fromResponse, err := s.transferService.AccountEnquiry(ctx, req.FromAccountNumber)
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
		toResponse, err := s.transferService.BeneficiaryEnquiry(ctx, req.ToAccountNumber, req.ToBankCode, req.TransferType)
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

	// Execute the transfer using the existing transfer service
	vfdResponse, err := s.transferService.InitiateTransfer(ctx, flowData.BusinessID, flowData.TransferRequest)
	if err != nil {
		processingTime := time.Since(startTime)
		return &domain.TransferFlowResult{
			Success:        false,
			Error:          err,
			ProcessingTime: processingTime,
		}, nil
	}

	processingTime := time.Since(startTime)

	// Get the transfer ID from the database (we need to query it since InitiateTransfer doesn't return it)
	// For now, we'll use 0 as placeholder - in a real implementation, you'd want to modify InitiateTransfer to return the ID
	transferID := uint(0) // TODO: Get actual transfer ID from database

	return &domain.TransferFlowResult{
		Success:        true,
		TransferID:     transferID,
		VFDResponse:    vfdResponse,
		ProcessingTime: processingTime,
	}, nil
}
