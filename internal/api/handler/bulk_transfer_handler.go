package handler

import (
	"encoding/json"
	"net/http"
	"payflow/internal/api/request"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/go-playground/validator/v10"
)

type BulkTransferHandler struct {
	bulkTransferService service.BulkTransferService
	validate            *validator.Validate
}

func NewBulkTransferHandler(bulkTransferService service.BulkTransferService) *BulkTransferHandler {
	return &BulkTransferHandler{
		bulkTransferService: bulkTransferService,
		validate:            validator.New(),
	}
}

// HandleSingleTransfer handles a single transfer request
func (h *BulkTransferHandler) HandleSingleTransfer(w http.ResponseWriter, r *http.Request) {
	var req request.BulkTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Get business ID from context (set by auth middleware)
	businessID, exists := r.Context().Value("business_id").(uint)
	if !exists {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	// Convert request to domain model
	bulkTransferReq := &domain.BulkTransferRequest{
		FromAccountNumber:  req.FromAccountNumber,
		ToAccountNumber:    req.ToAccountNumber,
		ToBankCode:         req.ToBankCode,
		Amount:             req.Amount,
		Remark:             req.Remark,
		TransferType:       req.TransferType,
		Reference:          req.Reference,
		FromAccountDetails: req.FromAccountDetails,
		ToAccountDetails:   req.ToAccountDetails,
		BankCode:           req.BankCode,
	}

	// Execute the transfer
	result, err := h.bulkTransferService.ExecuteSingleTransfer(r.Context(), businessID, bulkTransferReq)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Convert to response format
	transferResponse := response.BulkTransferResponse{
		Success:        result.Success,
		TransferID:     result.TransferID,
		Reference:      result.Reference,
		Error:          result.Error,
		ProcessingTime: result.ProcessingTime,
	}

	// Convert from account details
	if result.FromAccountDetails != nil {
		transferResponse.FromAccountDetails = &response.AccountEnquiryData{
			AccountNo:          result.FromAccountDetails.AccountNo,
			AccountBalance:     result.FromAccountDetails.AccountBalance,
			AccountId:          result.FromAccountDetails.AccountId,
			Client:             result.FromAccountDetails.Client,
			ClientId:           result.FromAccountDetails.ClientId,
			SavingsProductName: result.FromAccountDetails.SavingsProductName,
		}
	}

	// Convert to account details
	if result.ToAccountDetails != nil {
		transferResponse.ToAccountDetails = &response.BeneficiaryEnquiryData{
			Name:     result.ToAccountDetails.Name,
			ClientId: result.ToAccountDetails.ClientId,
			BVN:      result.ToAccountDetails.BVN,
			Account: struct {
				Number string `json:"number"`
				ID     string `json:"id"`
			}{
				Number: result.ToAccountDetails.Account.Number,
				ID:     result.ToAccountDetails.Account.ID,
			},
			Status:   result.ToAccountDetails.Status,
			Currency: result.ToAccountDetails.Currency,
			Bank:     result.ToAccountDetails.Bank,
		}
	}

	if result.VFDResponse != nil {
		transferResponse.VFDResponse = &response.TransferResponse{
			Status:  result.VFDResponse.Status,
			Message: result.VFDResponse.Message,
		}
		if result.VFDResponse.Data != nil {
			transferResponse.VFDResponse.Data = &response.TransferData{
				TxnId:     result.VFDResponse.Data.TxnId,
				SessionId: result.VFDResponse.Data.SessionId,
				Reference: result.VFDResponse.Data.Reference,
			}
		}
	}

	response.RespondWithJSON(w, http.StatusOK, transferResponse)
}

// HandleBatchTransfer handles a batch of transfer requests
func (h *BulkTransferHandler) HandleBatchTransfer(w http.ResponseWriter, r *http.Request) {
	var req request.BulkTransferBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Get business ID from context (set by auth middleware)
	businessID, exists := r.Context().Value("business_id").(uint)
	if !exists {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	// Convert request to domain model
	bulkTransferReqs := make([]domain.BulkTransferRequest, len(req.Transfers))
	for i, transferReq := range req.Transfers {
		bulkTransferReqs[i] = domain.BulkTransferRequest{
			FromAccountNumber:  transferReq.FromAccountNumber,
			ToAccountNumber:    transferReq.ToAccountNumber,
			ToBankCode:         transferReq.ToBankCode,
			Amount:             transferReq.Amount,
			Remark:             transferReq.Remark,
			TransferType:       transferReq.TransferType,
			Reference:          transferReq.Reference,
			FromAccountDetails: transferReq.FromAccountDetails,
			ToAccountDetails:   transferReq.ToAccountDetails,
			BankCode:           transferReq.BankCode,
		}
	}

	batchReq := &domain.BulkTransferBatchRequest{
		Transfers: bulkTransferReqs,
	}

	// Execute the batch transfer
	result, err := h.bulkTransferService.ExecuteBatchTransfer(r.Context(), businessID, batchReq)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Convert to response format
	transferResponses := make([]response.BulkTransferResponse, len(result.Transfers))
	for i, transferResult := range result.Transfers {
		transferResponses[i] = response.BulkTransferResponse{
			Success:        transferResult.Success,
			TransferID:     transferResult.TransferID,
			Reference:      transferResult.Reference,
			Error:          transferResult.Error,
			ProcessingTime: transferResult.ProcessingTime,
		}

		// Convert from account details
		if transferResult.FromAccountDetails != nil {
			transferResponses[i].FromAccountDetails = &response.AccountEnquiryData{
				AccountNo:          transferResult.FromAccountDetails.AccountNo,
				AccountBalance:     transferResult.FromAccountDetails.AccountBalance,
				AccountId:          transferResult.FromAccountDetails.AccountId,
				Client:             transferResult.FromAccountDetails.Client,
				ClientId:           transferResult.FromAccountDetails.ClientId,
				SavingsProductName: transferResult.FromAccountDetails.SavingsProductName,
			}
		}

		// Convert to account details
		if transferResult.ToAccountDetails != nil {
			transferResponses[i].ToAccountDetails = &response.BeneficiaryEnquiryData{
				Name:     transferResult.ToAccountDetails.Name,
				ClientId: transferResult.ToAccountDetails.ClientId,
				BVN:      transferResult.ToAccountDetails.BVN,
				Account: struct {
					Number string `json:"number"`
					ID     string `json:"id"`
				}{
					Number: transferResult.ToAccountDetails.Account.Number,
					ID:     transferResult.ToAccountDetails.Account.ID,
				},
				Status:   transferResult.ToAccountDetails.Status,
				Currency: transferResult.ToAccountDetails.Currency,
				Bank:     transferResult.ToAccountDetails.Bank,
			}
		}

		if transferResult.VFDResponse != nil {
			transferResponses[i].VFDResponse = &response.TransferResponse{
				Status:  transferResult.VFDResponse.Status,
				Message: transferResult.VFDResponse.Message,
			}
			if transferResult.VFDResponse.Data != nil {
				transferResponses[i].VFDResponse.Data = &response.TransferData{
					TxnId:     transferResult.VFDResponse.Data.TxnId,
					SessionId: transferResult.VFDResponse.Data.SessionId,
					Reference: transferResult.VFDResponse.Data.Reference,
				}
			}
		}
	}

	batchResponse := response.BulkTransferBatchResponse{
		TotalTransfers:      result.TotalTransfers,
		SuccessfulTransfers: result.SuccessfulTransfers,
		FailedTransfers:     result.FailedTransfers,
		Transfers:           transferResponses,
		ProcessingTime:      result.ProcessingTime,
	}

	response.RespondWithJSON(w, http.StatusOK, batchResponse)
}

// HandleGetTransferFlowData handles getting transfer flow data without executing the transfer
func (h *BulkTransferHandler) HandleGetTransferFlowData(w http.ResponseWriter, r *http.Request) {
	var req request.BulkTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Get business ID from context (set by auth middleware)
	businessID, exists := r.Context().Value("business_id").(uint)
	if !exists {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	// Convert request to domain model
	bulkTransferReq := &domain.BulkTransferRequest{
		FromAccountNumber:  req.FromAccountNumber,
		ToAccountNumber:    req.ToAccountNumber,
		ToBankCode:         req.ToBankCode,
		Amount:             req.Amount,
		Remark:             req.Remark,
		TransferType:       req.TransferType,
		Reference:          req.Reference,
		FromAccountDetails: req.FromAccountDetails,
		ToAccountDetails:   req.ToAccountDetails,
		BankCode:           req.BankCode,
	}

	// Get transfer flow data
	flowData, err := h.bulkTransferService.GetTransferFlowData(r.Context(), businessID, bulkTransferReq)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Convert to response format
	flowResponse := map[string]interface{}{
		"from_account_details": flowData.FromAccountDetails,
		"to_account_details":   flowData.ToAccountDetails,
		"transfer_request":     flowData.TransferRequest,
		"business_id":          flowData.BusinessID,
	}

	response.RespondWithJSON(w, http.StatusOK, flowResponse)
}
