package handler

import (
	"encoding/json"
	"net/http"

	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/go-chi/chi/v5"
)

// PlatformSettingsHandler handles platform settings endpoints (super admin only).
type PlatformSettingsHandler struct {
	settingsSvc service.PlatformSettingsService
}

// NewPlatformSettingsHandler creates a new platform settings handler.
func NewPlatformSettingsHandler(svc service.PlatformSettingsService) *PlatformSettingsHandler {
	return &PlatformSettingsHandler{settingsSvc: svc}
}

// HandleListSettings handles GET /platform/settings
func (h *PlatformSettingsHandler) HandleListSettings(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	var settings []service.SettingSummary
	var err error
	if category != "" {
		settings, err = h.settingsSvc.GetSettingsByCategory(r.Context(), category)
	} else {
		settings, err = h.settingsSvc.ListSettings(r.Context())
	}
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, settings)
}

// HandleSetSetting handles PUT /platform/settings/{key}
func (h *PlatformSettingsHandler) HandleSetSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	var req struct {
		Value       string `json:"value"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if req.Value == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.settingsSvc.SetSetting(r.Context(), key, req.Value, req.Description, req.Category); err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Setting updated successfully",
	})
}

// HandleDeleteSetting handles DELETE /platform/settings/{key}
func (h *PlatformSettingsHandler) HandleDeleteSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	if err := h.settingsSvc.DeleteSetting(r.Context(), key); err != nil {
		response.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
