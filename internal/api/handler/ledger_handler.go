package handler

import (
	"net/http"
	"strconv"

	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
)

// LedgerHandler handles ledger/accounting endpoints.
type LedgerHandler struct {
	ledgerSvc service.LedgerService
	walletSvc service.WalletService
}

// NewLedgerHandler creates a new ledger handler.
func NewLedgerHandler(ledgerSvc service.LedgerService, walletSvc service.WalletService) *LedgerHandler {
	return &LedgerHandler{ledgerSvc: ledgerSvc, walletSvc: walletSvc}
}

// HandleGetEntries handles GET /v1/wallets/ledger?page=1&limit=50
func (h *LedgerHandler) HandleGetEntries(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	entries, total, err := h.ledgerSvc.GetEntries(r.Context(), claims.BusinessID, page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// HandleReconcile handles GET /v1/wallets/reconcile
func (h *LedgerHandler) HandleReconcile(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	// Get current wallet balance
	balance, err := h.walletSvc.GetBalance(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	result, err := h.ledgerSvc.Reconcile(r.Context(), claims.BusinessID, balance)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, result)
}
