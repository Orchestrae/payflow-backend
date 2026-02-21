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
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type PayrollHandler struct {
	payrollService  service.PayrollService
	employeeService service.EmployeeService
	validate        *validator.Validate
}

func NewPayrollHandler(payrollSvc service.PayrollService, employeeSvc service.EmployeeService) *PayrollHandler {
	return &PayrollHandler{
		payrollService:  payrollSvc,
		employeeService: employeeSvc,
		validate:        validator.New(),
	}
}

// CreatePayrollRun handles POST /payroll-runs
func (h *PayrollHandler) CreatePayrollRun(w http.ResponseWriter, r *http.Request) {
	var req request.CreatePayrollRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	// Parse period - default to current month if not provided
	var period time.Time
	if req.Period != "" {
		// Try parsing as "YYYY-MM" or "YYYY-MM-DD"
		parsed, err := time.Parse("2006-01", req.Period)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", req.Period)
			if err != nil {
				response.RespondWithError(w, domain.ErrValidationFailed)
				return
			}
		}
		period = parsed
	} else {
		// Default to current month
		now := time.Now()
		period = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	// Convert map[string][]AdjustmentItem to map[uint][]EmployeeAdjustment for the service layer
	// The key can be employee ID (as string) or email
	adjustments := make(map[uint][]service.EmployeeAdjustment)
	if req.Adjustments != nil {
		// Get all employees for this business to resolve identifiers
		allEmployeesList, err := h.employeeService.ListByBusinessID(r.Context(), claims.BusinessID)
		// Convert []domain.Employee to []*domain.Employee for easier lookup
		allEmployees := make([]*domain.Employee, len(allEmployeesList))
		for i := range allEmployeesList {
			allEmployees[i] = &allEmployeesList[i]
		}
		
		// Create lookup maps: email -> employee ID, and ID string -> employee ID
		emailToID := make(map[string]uint)
		idStrToID := make(map[string]uint)
		if err == nil {
			for _, emp := range allEmployees {
				emailToID[emp.Email] = emp.ID
				idStrToID[strconv.FormatUint(uint64(emp.ID), 10)] = emp.ID
			}
		}

		for identifier, items := range req.Adjustments {
			var employeeID uint
			var found bool

			// Try to parse as employee ID first
			if parsedID, err := strconv.ParseUint(identifier, 10, 32); err == nil {
				if id, ok := idStrToID[identifier]; ok {
					employeeID = id
					found = true
				} else {
					// If not in lookup, assume it's a valid ID (for cases where employee list might be incomplete)
					employeeID = uint(parsedID)
					found = true
				}
			} else {
				// Try to resolve as email
				if id, ok := emailToID[identifier]; ok {
					employeeID = id
					found = true
				}
			}

			if found {
				// Convert request adjustment items to service adjustment items
				serviceAdjustments := make([]service.EmployeeAdjustment, len(items))
				for i, item := range items {
					serviceAdjustments[i] = service.EmployeeAdjustment{
						ItemName:      item.ItemName,
						Amount:        item.Amount,
						Description:   item.Description,
						ComponentType: item.ComponentType,
					}
				}
				adjustments[employeeID] = serviceAdjustments
			}
		}
	}

	payrollRun, err := h.payrollService.CreateAndStorePayrollRun(r.Context(), claims.BusinessID, period, adjustments)
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

	var req request.RejectPayrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	rejectedRun, err := h.payrollService.RejectPayrollRun(r.Context(), uint(runID), claims.UserID, req.Reason)
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

// ProcessPayrollRunInstantly handles POST /payroll-runs/{runID}/process-now
// This endpoint processes a payroll run immediately, bypassing the scheduler.
// Allows businesses to pay employees instantly without waiting for scheduled processing.
func (h *PayrollHandler) ProcessPayrollRunInstantly(w http.ResponseWriter, r *http.Request) {
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

	updatedRun, err := h.payrollService.ProcessPayrollRunInstantly(r.Context(), uint(runID), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, updatedRun)
}

// UpdateBusinessPayrollConfig handles PATCH /payroll-runs/config
// Updates the business payroll workflow configuration
func (h *PayrollHandler) UpdateBusinessPayrollConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequiresApproval *bool `json:"requires_approval,omitempty"`
		AutoProcess      *bool `json:"auto_process,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	// This would require a business service method to update config
	// For now, we'll add it to the payroll service
	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Configuration update endpoint - to be implemented",
		"business_id": claims.BusinessID,
	})
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
