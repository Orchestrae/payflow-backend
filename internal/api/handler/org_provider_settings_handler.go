package handler

import (
	"encoding/json"
	"net/http"

	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/go-chi/chi/v5"
)

// OrgProviderSettingsHandler handles org-level provider key overrides.
type OrgProviderSettingsHandler struct {
	svc service.OrgProviderSettingsService
}

func NewOrgProviderSettingsHandler(svc service.OrgProviderSettingsService) *OrgProviderSettingsHandler {
	return &OrgProviderSettingsHandler{svc: svc}
}

// HandleListOrgKeys handles GET /v1/business/provider-keys
func (h *OrgProviderSettingsHandler) HandleListOrgKeys(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	settings, err := h.svc.ListSettings(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, settings)
}

// HandleSetOrgKey handles PUT /v1/business/provider-keys/{provider}/{key}
func (h *OrgProviderSettingsHandler) HandleSetOrgKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	provider := chi.URLParam(r, "provider")
	key := chi.URLParam(r, "key")

	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Value == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Validate provider and key names
	validProviders := map[string]map[string]bool{
		"paystack": {
			domain.OrgSettingPaystackSecretKey: true,
			domain.OrgSettingPaystackPublicKey: true,
		},
		"korapay": {
			domain.OrgSettingKorapayAPIKey: true,
			domain.OrgSettingKorapayEncKey: true,
		},
	}

	providerKeys, ok := validProviders[provider]
	if !ok {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}
	if !providerKeys[key] {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.svc.SetKey(r.Context(), claims.BusinessID, provider, key, req.Value); err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Provider key saved successfully",
	})
}

// HandleDeleteOrgKey handles DELETE /v1/business/provider-keys/{provider}/{key}
func (h *OrgProviderSettingsHandler) HandleDeleteOrgKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	provider := chi.URLParam(r, "provider")
	key := chi.URLParam(r, "key")

	if err := h.svc.DeleteKey(r.Context(), claims.BusinessID, provider, key); err != nil {
		response.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
