package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"payflow/internal/api/middleware"
	"payflow/internal/api/request"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
)

// TransferHandler handles transfer HTTP requests
type TransferHandler struct {
	transferService service.TransferService
	validate        *validator.Validate
}

// NewTransferHandler creates a new transfer handler
func NewTransferHandler(transferService service.TransferService) *TransferHandler {
	return &TransferHandler{
		transferService: transferService,
		validate:        validator.New(),
	}
}

// HandleSingleTransfer handles a single transfer request
// POST /v1/transfers
func (h *TransferHandler) HandleSingleTransfer(w http.ResponseWriter, r *http.Request) {
	var req request.SingleTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Get business ID from context (set by auth middleware)
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	// Convert to domain request - simple mapping, service handles the rest
	domainReq := &domain.SingleTransferRequest{
		Reference:         req.Reference,
		Amount:            req.Amount,
		BankCode:          req.BankCode,
		AccountNumber:     req.AccountNumber,
		AccountName:       req.AccountName,
		Narration:         req.Narration,
		PreferredProvider: domain.ProviderName(req.Provider),
	}

	// Execute the transfer
	result, err := h.transferService.ExecuteTransfer(r.Context(), claims.BusinessID, domainReq)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Convert to response format
	transferResponse := response.SingleTransferResponse{
		Success:        result.Success,
		TransferID:     result.TransferID,
		Reference:      result.Reference,
		TransactionID:  result.TransactionID,
		Status:         result.Status,
		Message:        result.Message,
		Provider:       string(result.Provider),
		Currency:       "NGN",
		Fee:            result.Fee,
		ProcessingTime: result.ProcessingTime.String(),
		Error:          result.Error,
	}

	response.RespondWithJSON(w, http.StatusOK, transferResponse)
}

// HandleBatchTransfer handles a batch of transfer requests
// POST /v1/transfers/batch
func (h *TransferHandler) HandleBatchTransfer(w http.ResponseWriter, r *http.Request) {
	var req request.BatchTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Get business ID from context (set by auth middleware)
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	// Convert request to domain model
	transfers := make([]domain.SingleTransferRequest, len(req.Transfers))
	for i, t := range req.Transfers {
		transfers[i] = domain.SingleTransferRequest{
			Reference:     t.Reference, // Optional - service will generate if empty
			Amount:        t.Amount,
			BankCode:      t.BankCode,
			AccountNumber: t.AccountNumber,
			AccountName:   t.AccountName,
			Narration:     t.Narration,
		}
	}

	domainReq := &domain.BulkTransferRequest{
		Transfers: transfers,
	}

	// Execute the batch transfer
	result, err := h.transferService.ExecuteBatchTransfer(r.Context(), claims.BusinessID, domainReq)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Convert to response format
	transferResponses := make([]response.SingleTransferResponse, len(result.Transfers))
	for i, t := range result.Transfers {
		transferResponses[i] = response.SingleTransferResponse{
			Success:        t.Success,
			TransferID:     t.TransferID,
			Reference:      t.Reference,
			TransactionID:  t.TransactionID,
			Status:         t.Status,
			Message:        t.Message,
			Provider:       string(t.Provider),
			Fee:            t.Fee,
			ProcessingTime: t.ProcessingTime.String(),
			Error:          t.Error,
		}
	}

	batchResponse := response.BatchTransferResponse{
		TotalTransfers:      result.TotalTransfers,
		SuccessfulTransfers: result.SuccessfulTransfers,
		FailedTransfers:     result.FailedTransfers,
		Transfers:           transferResponses,
		ProcessingTime:      result.ProcessingTime.String(),
	}

	response.RespondWithJSON(w, http.StatusOK, batchResponse)
}

// HandleGetTransfer gets a specific transfer by ID
// GET /v1/transfers/{id}
func (h *TransferHandler) HandleGetTransfer(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	transfer, err := h.transferService.GetTransferByID(r.Context(), uint(id))
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, transfer)
}

// HandleListTransfers lists transfers for a business
// GET /v1/transfers
func (h *TransferHandler) HandleListTransfers(w http.ResponseWriter, r *http.Request) {
	// Get business ID from context
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}
	businessID := claims.BusinessID

	// Parse pagination
	page := 1
	limit := 20

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	transfers, total, err := h.transferService.ListTransfers(r.Context(), businessID, page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"transfers": transfers,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// HandleRetryTransfer retries a failed transfer
// POST /v1/transfers/{id}/retry
func (h *TransferHandler) HandleRetryTransfer(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	result, err := h.transferService.RetryTransfer(r.Context(), claims.BusinessID, uint(id))
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	transferResponse := response.SingleTransferResponse{
		Success:       result.Success,
		TransferID:    result.TransferID,
		Reference:     result.Reference,
		TransactionID: result.TransactionID,
		Status:        result.Status,
		Message:       result.Message,
		Provider:      string(result.Provider),
		Currency:      "NGN",
		Fee:           result.Fee,
	}

	response.RespondWithJSON(w, http.StatusOK, transferResponse)
}
