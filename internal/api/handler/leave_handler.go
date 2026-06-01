package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/go-chi/chi/v5"
)

type LeaveHandler struct {
	leaveSvc service.LeaveService
}

func NewLeaveHandler(svc service.LeaveService) *LeaveHandler {
	return &LeaveHandler{leaveSvc: svc}
}

// HandleCreateLeaveType handles POST /v1/leave/types
func (h *LeaveHandler) HandleCreateLeaveType(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	var req struct {
		Name             string `json:"name"`
		DefaultDays      int    `json:"default_days"`
		RequiresApproval *bool  `json:"requires_approval"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.DefaultDays <= 0 {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	lt := &domain.LeaveType{
		BusinessID:       claims.BusinessID,
		Name:             req.Name,
		DefaultDays:      req.DefaultDays,
		RequiresApproval: true,
	}
	if req.RequiresApproval != nil {
		lt.RequiresApproval = *req.RequiresApproval
	}

	result, err := h.leaveSvc.CreateLeaveType(r.Context(), lt)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusCreated, result)
}

// HandleListLeaveTypes handles GET /v1/leave/types
func (h *LeaveHandler) HandleListLeaveTypes(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	types, err := h.leaveSvc.ListLeaveTypes(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusOK, types)
}

// HandleSubmitRequest handles POST /v1/leave/requests
func (h *LeaveHandler) HandleSubmitRequest(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	var req domain.LeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}
	req.BusinessID = claims.BusinessID

	result, err := h.leaveSvc.SubmitRequest(r.Context(), &req)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusCreated, result)
}

// HandleListRequests handles GET /v1/leave/requests
func (h *LeaveHandler) HandleListRequests(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	requests, total, err := h.leaveSvc.ListRequests(r.Context(), claims.BusinessID, page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"requests": requests,
		"total":    total,
	})
}

// HandleApproveRequest handles POST /v1/leave/requests/{id}/approve
func (h *LeaveHandler) HandleApproveRequest(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.leaveSvc.ApproveRequest(r.Context(), uint(id), claims.UserID); err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Leave approved"})
}

// HandleRejectRequest handles POST /v1/leave/requests/{id}/reject
func (h *LeaveHandler) HandleRejectRequest(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.leaveSvc.RejectRequest(r.Context(), uint(id), claims.UserID, req.Reason); err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Leave rejected"})
}

// HandleGetBalances handles GET /v1/leave/balances/{employeeID}
func (h *LeaveHandler) HandleGetBalances(w http.ResponseWriter, r *http.Request) {
	empID, err := strconv.ParseUint(chi.URLParam(r, "employeeID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	balances, err := h.leaveSvc.GetBalance(r.Context(), uint(empID), year)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusOK, balances)
}
