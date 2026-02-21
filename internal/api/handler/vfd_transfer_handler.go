package handler

import (
	"encoding/json"
	"net/http"
	"payflow/internal/api/middleware"
	"payflow/internal/api/request"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type VFDTransferHandler struct {
	transferService service.VFDTransferService
	validate        *validator.Validate
}

func NewVFDTransferHandler(transferService service.VFDTransferService) *VFDTransferHandler {
	return &VFDTransferHandler{
		transferService: transferService,
		validate:        validator.New(),
	}
}

// HandleAccountEnquiry handles account enquiry requests
func (h *VFDTransferHandler) HandleAccountEnquiry(w http.ResponseWriter, r *http.Request) {
	var req request.AccountEnquiryRequest

	// Parse query parameters
	accountNumber := r.URL.Query().Get("accountNumber")
	req.AccountNumber = accountNumber

	accountResponse, err := h.transferService.AccountEnquiry(r.Context(), req.AccountNumber)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, accountResponse)
}

// HandleBeneficiaryEnquiry handles beneficiary enquiry requests
func (h *VFDTransferHandler) HandleBeneficiaryEnquiry(w http.ResponseWriter, r *http.Request) {
	var req request.BeneficiaryEnquiryRequest

	// Parse query parameters
	req.AccountNo = r.URL.Query().Get("accountNo")
	req.Bank = r.URL.Query().Get("bank")
	req.TransferType = r.URL.Query().Get("transfer_type")

	if req.AccountNo == "" || req.Bank == "" || req.TransferType == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	beneficiaryResponse, err := h.transferService.BeneficiaryEnquiry(r.Context(), req.AccountNo, req.Bank, req.TransferType)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, beneficiaryResponse)
}

// HandleGetBankList handles bank list requests
func (h *VFDTransferHandler) HandleGetBankList(w http.ResponseWriter, r *http.Request) {
	bankListResponse, err := h.transferService.GetBankList(r.Context())
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, bankListResponse)
}

// HandleInitiateTransfer handles transfer initiation requests
func (h *VFDTransferHandler) HandleInitiateTransfer(w http.ResponseWriter, r *http.Request) {
	var req request.VFDTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}
	businessID := claims.BusinessID

	transferReq := &domain.TransferRequest{
		FromAccount:           req.FromAccount,
		UniqueSenderAccountId: req.UniqueSenderAccountId,
		FromClientId:          req.FromClientId,
		FromClient:            req.FromClient,
		FromSavingsId:         req.FromSavingsId,
		FromBvn:               req.FromBvn,
		ToClientId:            req.ToClientId,
		ToClient:              req.ToClient,
		ToSavingsId:           req.ToSavingsId,
		ToSession:             req.ToSession,
		ToBvn:                 req.ToBvn,
		ToAccount:             req.ToAccount,
		ToBank:                req.ToBank,
		Signature:             req.Signature,
		Amount:                req.Amount,
		Remark:                req.Remark,
		TransferType:          req.TransferType,
		Reference:             req.Reference,
	}

	transferResponse, err := h.transferService.InitiateTransfer(r.Context(), businessID, transferReq)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, transferResponse)
}

// HandleListTransfers handles listing transfer records
func (h *VFDTransferHandler) HandleListTransfers(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}
	businessID := claims.BusinessID

	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		pageStr = "1"
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "20"
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	transfers, total, err := h.transferService.ListTransfers(r.Context(), businessID, page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Convert to response format
	transferResponses := make([]response.TransferRecordResponse, len(transfers))
	for i, transfer := range transfers {
		transferResponses[i] = response.TransferRecordResponse{
			ID:              transfer.ID,
			BusinessID:      transfer.BusinessID,
			FromAccount:     transfer.FromAccount,
			FromClientId:    transfer.FromClientId,
			FromClient:      transfer.FromClient,
			FromSavingsId:   transfer.FromSavingsId,
			FromBvn:         transfer.FromBvn,
			ToClientId:      transfer.ToClientId,
			ToClient:        transfer.ToClient,
			ToSavingsId:     transfer.ToSavingsId,
			ToSession:       transfer.ToSession,
			ToBvn:           transfer.ToBvn,
			ToAccount:       transfer.ToAccount,
			ToBank:          transfer.ToBank,
			Amount:          transfer.Amount,
			Remark:          transfer.Remark,
			TransferType:    transfer.TransferType,
			Reference:       transfer.Reference,
			TxnId:           transfer.TxnId,
			SessionId:       transfer.SessionId,
			Status:          transfer.Status,
			VFDStatus:       transfer.VFDStatus,
			VFDMessage:      transfer.VFDMessage,
			ProcessedAt:     transfer.ProcessedAt,
			ProcessingError: transfer.ProcessingError,
			CreatedAt:       transfer.CreatedAt,
		}
	}

	listResponse := response.TransferListResponse{
		Transfers: transferResponses,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}

	response.RespondWithJSON(w, http.StatusOK, listResponse)
}

// HandleGetTransferByID handles getting a specific transfer by ID
func (h *VFDTransferHandler) HandleGetTransferByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

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

	// Convert to response format
	transferResponse := response.TransferRecordResponse{
		ID:              transfer.ID,
		BusinessID:      transfer.BusinessID,
		FromAccount:     transfer.FromAccount,
		FromClientId:    transfer.FromClientId,
		FromClient:      transfer.FromClient,
		FromSavingsId:   transfer.FromSavingsId,
		FromBvn:         transfer.FromBvn,
		ToClientId:      transfer.ToClientId,
		ToClient:        transfer.ToClient,
		ToSavingsId:     transfer.ToSavingsId,
		ToSession:       transfer.ToSession,
		ToBvn:           transfer.ToBvn,
		ToAccount:       transfer.ToAccount,
		ToBank:          transfer.ToBank,
		Amount:          transfer.Amount,
		Remark:          transfer.Remark,
		TransferType:    transfer.TransferType,
		Reference:       transfer.Reference,
		TxnId:           transfer.TxnId,
		SessionId:       transfer.SessionId,
		Status:          transfer.Status,
		VFDStatus:       transfer.VFDStatus,
		VFDMessage:      transfer.VFDMessage,
		ProcessedAt:     transfer.ProcessedAt,
		ProcessingError: transfer.ProcessingError,
		CreatedAt:       transfer.CreatedAt,
	}

	response.RespondWithJSON(w, http.StatusOK, transferResponse)
}

// HandleGetTransfersByFromAccount handles getting transfers by from account
func (h *VFDTransferHandler) HandleGetTransfersByFromAccount(w http.ResponseWriter, r *http.Request) {
	fromAccount := r.URL.Query().Get("fromAccount")
	if fromAccount == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		pageStr = "1"
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "20"
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	transfers, total, err := h.transferService.GetTransfersByFromAccount(r.Context(), fromAccount, page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Convert to response format
	transferResponses := make([]response.TransferRecordResponse, len(transfers))
	for i, transfer := range transfers {
		transferResponses[i] = response.TransferRecordResponse{
			ID:              transfer.ID,
			BusinessID:      transfer.BusinessID,
			FromAccount:     transfer.FromAccount,
			FromClientId:    transfer.FromClientId,
			FromClient:      transfer.FromClient,
			FromSavingsId:   transfer.FromSavingsId,
			FromBvn:         transfer.FromBvn,
			ToClientId:      transfer.ToClientId,
			ToClient:        transfer.ToClient,
			ToSavingsId:     transfer.ToSavingsId,
			ToSession:       transfer.ToSession,
			ToBvn:           transfer.ToBvn,
			ToAccount:       transfer.ToAccount,
			ToBank:          transfer.ToBank,
			Amount:          transfer.Amount,
			Remark:          transfer.Remark,
			TransferType:    transfer.TransferType,
			Reference:       transfer.Reference,
			TxnId:           transfer.TxnId,
			SessionId:       transfer.SessionId,
			Status:          transfer.Status,
			VFDStatus:       transfer.VFDStatus,
			VFDMessage:      transfer.VFDMessage,
			ProcessedAt:     transfer.ProcessedAt,
			ProcessingError: transfer.ProcessingError,
			CreatedAt:       transfer.CreatedAt,
		}
	}

	listResponse := response.TransferListResponse{
		Transfers: transferResponses,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}

	response.RespondWithJSON(w, http.StatusOK, listResponse)
}

// HandleGetTransfersByToAccount handles getting transfers by to account
func (h *VFDTransferHandler) HandleGetTransfersByToAccount(w http.ResponseWriter, r *http.Request) {
	toAccount := r.URL.Query().Get("toAccount")
	if toAccount == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		pageStr = "1"
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "20"
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	transfers, total, err := h.transferService.GetTransfersByToAccount(r.Context(), toAccount, page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Convert to response format
	transferResponses := make([]response.TransferRecordResponse, len(transfers))
	for i, transfer := range transfers {
		transferResponses[i] = response.TransferRecordResponse{
			ID:              transfer.ID,
			BusinessID:      transfer.BusinessID,
			FromAccount:     transfer.FromAccount,
			FromClientId:    transfer.FromClientId,
			FromClient:      transfer.FromClient,
			FromSavingsId:   transfer.FromSavingsId,
			FromBvn:         transfer.FromBvn,
			ToClientId:      transfer.ToClientId,
			ToClient:        transfer.ToClient,
			ToSavingsId:     transfer.ToSavingsId,
			ToSession:       transfer.ToSession,
			ToBvn:           transfer.ToBvn,
			ToAccount:       transfer.ToAccount,
			ToBank:          transfer.ToBank,
			Amount:          transfer.Amount,
			Remark:          transfer.Remark,
			TransferType:    transfer.TransferType,
			Reference:       transfer.Reference,
			TxnId:           transfer.TxnId,
			SessionId:       transfer.SessionId,
			Status:          transfer.Status,
			VFDStatus:       transfer.VFDStatus,
			VFDMessage:      transfer.VFDMessage,
			ProcessedAt:     transfer.ProcessedAt,
			ProcessingError: transfer.ProcessingError,
			CreatedAt:       transfer.CreatedAt,
		}
	}

	listResponse := response.TransferListResponse{
		Transfers: transferResponses,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}

	response.RespondWithJSON(w, http.StatusOK, listResponse)
}
