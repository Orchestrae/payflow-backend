package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service/platform"
)

// PlatformHandler handles super admin platform endpoints.
type PlatformHandler struct {
	platformService platform.PlatformService
}

// NewPlatformHandler creates a new platform handler.
func NewPlatformHandler(svc platform.PlatformService) *PlatformHandler {
	return &PlatformHandler{platformService: svc}
}

// GetStats handles GET /platform/stats
func (h *PlatformHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.platformService.GetStats(r.Context())
	if err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusOK, stats)
}

// ListOrganizations handles GET /platform/organizations
func (h *PlatformHandler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	orgs, total, err := h.platformService.ListOrganizations(r.Context(), page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data":  orgs,
		"total": total,
	})
}

// SuspendOrganization handles POST /platform/organizations/{id}/suspend
func (h *PlatformHandler) SuspendOrganization(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if err := h.platformService.SuspendOrganization(r.Context(), uint(id), body.Reason); err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Organization suspended"})
}

// ActivateOrganization handles POST /platform/organizations/{id}/activate
func (h *PlatformHandler) ActivateOrganization(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.platformService.ActivateOrganization(r.Context(), uint(id)); err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Organization activated"})
}
