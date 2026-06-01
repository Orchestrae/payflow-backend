package handler

import (
	"net/http"

	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
)

// VerificationHandler handles bank account verification endpoints.
type VerificationHandler struct {
	verificationService service.AccountVerificationService
}

// NewVerificationHandler creates a new verification handler.
func NewVerificationHandler(svc service.AccountVerificationService) *VerificationHandler {
	return &VerificationHandler{verificationService: svc}
}

// HandleVerifyBankAccount handles GET /v1/verify/bank-account?bank_code=058&account_number=0123456789
func (h *VerificationHandler) HandleVerifyBankAccount(w http.ResponseWriter, r *http.Request) {
	bankCode := r.URL.Query().Get("bank_code")
	accountNumber := r.URL.Query().Get("account_number")

	if bankCode == "" || accountNumber == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	result, err := h.verificationService.VerifyBankAccount(r.Context(), bankCode, accountNumber)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, result)
}

// HandleVerifyBVN handles POST /v1/verify/bvn
func (h *VerificationHandler) HandleVerifyBVN(w http.ResponseWriter, r *http.Request) {
	bvn := r.URL.Query().Get("bvn")
	if bvn == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	result, err := h.verificationService.VerifyBVN(r.Context(), bvn)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, result)
}
