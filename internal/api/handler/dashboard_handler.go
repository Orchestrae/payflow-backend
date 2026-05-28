package handler

import (
	"net/http"

	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
)

// DashboardHandler handles dashboard summary endpoints.
type DashboardHandler struct {
	dashboardService service.DashboardService
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(svc service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: svc}
}

// GetSummary handles GET /v1/dashboard
func (h *DashboardHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	summary, err := h.dashboardService.GetSummary(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, summary)
}
