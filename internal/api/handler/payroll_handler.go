// internal/api/handler/payroll_handler.go
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
	"github.com/go-playground/validator/v10"
)

type PayrollHandler struct {
	payrollService service.PayrollService
	validate       *validator.Validate
}

func NewPayrollHandler(svc service.PayrollService) *PayrollHandler {
	return &PayrollHandler{
		payrollService: svc,
		validate:       validator.New(),
	}
}

// CreatePayrollRun handles POST /payroll-runs
func (h *PayrollHandler) CreatePayrollRun(w http.ResponseWriter, r *http.Request) {
	var req request.CreatePayrollRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	// Convert map[string]int64 to map[uint]int64 for the service layer
	adjustments := make(map[uint]int64)
	for empIDStr, amount := range req.Adjustments {
		empID, err := strconv.ParseUint(empIDStr, 10, 32)
		if err == nil {
			adjustments[uint(empID)] = amount
		}
	}

	payrollRun, err := h.payrollService.CreateAndStorePayrollRun(r.Context(), claims.BusinessID, adjustments)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, payrollRun)
}

// SubmitForApproval handles POST /payroll-runs/{runID}/submit
func (h *PayrollHandler) SubmitForApproval(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.ParseUint(chi.URLParam(r, "runID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	updatedRun, err := h.payrollService.SubmitForApproval(r.Context(), uint(runID), claims.UserID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, updatedRun)
}

// ApprovePayrollRun handles POST /payroll-runs/{runID}/approve
func (h *PayrollHandler) ApprovePayrollRun(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.ParseUint(chi.URLParam(r, "runID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	updatedRun, err := h.payrollService.ApprovePayrollRun(r.Context(), uint(runID), claims.UserID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, updatedRun)
}

func (h *PayrollHandler) RejectPayrollRun(w http.ResponseWriter, r *http.Request) {
	// Get runID from URL
	runIDStr := chi.URLParam(r, "runID")
	runID, _ := strconv.ParseUint(runIDStr, 10, 32)

	// Get user claims from context
	claims := r.Context().Value(middleware.UserClaimsKey).(*utils.Claims)
	rejecterID, _ := strconv.ParseUint(claims.UserID, 10, 32)

	// Decode request body
	var req request.RejectPayrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Call the service method
	rejectedRun, err := h.payrollService.RejectPayrollRun(r.Context(), uint(runID), uint(rejecterID), req.Reason)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, rejectedRun)
}

// ListPayrollRuns handles GET /payroll-runs
func (h *PayrollHandler) ListPayrollRuns(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	runs, err := h.payrollService.ListByBusinessID(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, runs)
}

// GetPayrollRunByID handles GET /payroll-runs/{runID}
func (h *PayrollHandler) GetPayrollRunByID(w http.ResponseWriter, r *http.Request) {
	runID, err := strconv.ParseUint(chi.URLParam(r, "runID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	run, err := h.payrollService.GetByID(r.Context(), uint(runID), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, run)
}
