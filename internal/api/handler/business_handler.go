package handler

import (
	"encoding/json"
	"net/http"

	"payflow/internal/api/middleware"
	"payflow/internal/api/request"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
)

// BusinessHandler handles business settings endpoints.
type BusinessHandler struct {
	businessService service.BusinessService
}

// NewBusinessHandler creates a new business handler.
func NewBusinessHandler(svc service.BusinessService) *BusinessHandler {
	return &BusinessHandler{businessService: svc}
}

// GetSettings handles GET /v1/business/settings
func (h *BusinessHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	business, err := h.businessService.GetSettings(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, business)
}

// UpdateSettings handles PATCH /v1/business/settings
func (h *BusinessHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req request.UpdateBusinessSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	// Build update map from non-nil fields
	updates := make(map[string]interface{})
	if req.PensionEnabled != nil {
		updates["pension_enabled"] = *req.PensionEnabled
	}
	if req.NHFEnabled != nil {
		updates["nhf_enabled"] = *req.NHFEnabled
	}
	if req.NSITFEnabled != nil {
		updates["nsitf_enabled"] = *req.NSITFEnabled
	}
	if req.PAYEEnabled != nil {
		updates["paye_enabled"] = *req.PAYEEnabled
	}
	if req.PayrollRequiresApproval != nil {
		updates["payroll_requires_approval"] = *req.PayrollRequiresApproval
	}
	if req.PayrollAutoProcess != nil {
		updates["payroll_auto_process"] = *req.PayrollAutoProcess
	}

	if len(updates) == 0 {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	business, err := h.businessService.UpdateSettings(r.Context(), claims.BusinessID, updates)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, business)
}
