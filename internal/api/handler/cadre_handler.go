// internal/api/handler/cadre_handler.go
package handler

import (
	"encoding/json"
	"net/http"
	"payflow/internal/api/middleware"
	"payflow/internal/api/request"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
	"payflow/pkg/utils"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type CadreHandler struct {
	cadreService service.CadreService
}

func NewCadreHandler(svc service.CadreService) *CadreHandler {
	return &CadreHandler{cadreService: svc}
}

// CreateCadre handles POST /cadres
func (h *CadreHandler) CreateCadre(w http.ResponseWriter, r *http.Request) {
	var req request.CreateCadreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims := r.Context().Value(middleware.UserClaimsKey).(*utils.Claims)
	businessID, _ := strconv.ParseUint(claims.BusinessID, 10, 32)

	// Create the cadre
	cadre := &domain.Cadre{
		BusinessID: uint(businessID),
		Name:       req.Name,
	}

	createdCadre, err := h.cadreService.CreateCadre(r.Context(), cadre)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, createdCadre)
}

// ListCadres handles GET /cadres
func (h *CadreHandler) ListCadres(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserClaimsKey).(*utils.Claims)
	businessID, _ := strconv.ParseUint(claims.BusinessID, 10, 32)

	cadres, err := h.cadreService.ListByBusinessID(r.Context(), uint(businessID))
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, cadres)
}

// GetCadreByID handles GET /cadres/{cadreID}
func (h *CadreHandler) GetCadreByID(w http.ResponseWriter, r *http.Request) {
	cadreIDStr := chi.URLParam(r, "cadreID")
	cadreID, _ := strconv.ParseUint(cadreIDStr, 10, 32)

	claims := r.Context().Value(middleware.UserClaimsKey).(*utils.Claims)
	businessID, _ := strconv.ParseUint(claims.BusinessID, 10, 32)

	cadre, err := h.cadreService.GetByID(r.Context(), uint(cadreID), uint(businessID))
	if err != nil {
		response.RespondWithError(w, err) // Service layer should return domain.ErrNotFound or domain.ErrForbidden
		return
	}

	response.RespondWithJSON(w, http.StatusOK, cadre)
}

// UpdateCadre handles PUT /cadres/{cadreID}
func (h *CadreHandler) UpdateCadre(w http.ResponseWriter, r *http.Request) {
	// cadreIDStr := chi.URLParam(r, "cadreID")
	// cadreID, _ := strconv.ParseUint(cadreIDStr, 10, 32)

	var req request.UpdateCadreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims := r.Context().Value(middleware.UserClaimsKey).(*utils.Claims)
	businessID, _ := strconv.ParseUint(claims.BusinessID, 10, 32)

	// Update the cadre
	cadre := &domain.Cadre{
		BusinessID: uint(businessID),
		Name:       req.Name,
	}

	updatedCadre, err := h.cadreService.UpdateCadre(r.Context(), cadre)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, updatedCadre)
}

// DeleteCadre handles DELETE /cadres/{cadreID}
func (h *CadreHandler) DeleteCadre(w http.ResponseWriter, r *http.Request) {
	cadreIDStr := chi.URLParam(r, "cadreID")
	cadreID, _ := strconv.ParseUint(cadreIDStr, 10, 32)

	claims := r.Context().Value(middleware.UserClaimsKey).(*utils.Claims)
	businessID, _ := strconv.ParseUint(claims.BusinessID, 10, 32)

	// Delete the cadre
	if err := h.cadreService.DeleteCadre(r.Context(), uint(cadreID), uint(businessID)); err != nil {
		response.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
