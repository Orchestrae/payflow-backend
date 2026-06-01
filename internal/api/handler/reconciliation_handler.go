package handler

import (
	"net/http"

	"payflow/internal/api/response"
	"payflow/internal/service"
)

// ReconciliationHandler handles provider reconciliation endpoints.
type ReconciliationHandler struct {
	reconciliationSvc service.ReconciliationService
}

// NewReconciliationHandler creates a new reconciliation handler.
func NewReconciliationHandler(reconciliationSvc service.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{reconciliationSvc: reconciliationSvc}
}

// HandleProviderReconciliation handles GET /platform/reconciliation/provider
// Manually triggers a provider reconciliation and returns the result.
func (h *ReconciliationHandler) HandleProviderReconciliation(w http.ResponseWriter, r *http.Request) {
	result, err := h.reconciliationSvc.RunProviderReconciliation(r.Context())
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, result)
}
