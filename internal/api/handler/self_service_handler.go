package handler

import (
	"encoding/json"
	"net/http"

	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
)

// SelfServiceHandler handles employee self-service endpoints.
type SelfServiceHandler struct {
	employeeService service.EmployeeService
	payrollService  service.PayrollService
}

// NewSelfServiceHandler creates a new self-service handler.
func NewSelfServiceHandler(empSvc service.EmployeeService, payrollSvc service.PayrollService) *SelfServiceHandler {
	return &SelfServiceHandler{employeeService: empSvc, payrollService: payrollSvc}
}

// GetProfile handles GET /v1/me/profile
func (h *SelfServiceHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	// Find employee linked to this user
	employees, err := h.employeeService.ListByBusinessID(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Find employee linked to this user
	for _, emp := range employees {
		if emp.UserID != nil && *emp.UserID == claims.UserID {
			response.RespondWithJSON(w, http.StatusOK, emp)
			return
		}
	}

	response.RespondWithError(w, domain.ErrNotFound)
}

type updateBankDetailsRequest struct {
	BankName          string `json:"bank_name"`
	BankCode          string `json:"bank_code"`
	BankAccountNumber string `json:"bank_account_number"`
}

// UpdateBankDetails handles PATCH /v1/me/bank-details
func (h *SelfServiceHandler) UpdateBankDetails(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	var req updateBankDetailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Find employee for this user
	employees, _ := h.employeeService.ListByBusinessID(r.Context(), claims.BusinessID)
	var employee *domain.Employee
	for i := range employees {
		if employees[i].UserID != nil && *employees[i].UserID == claims.UserID {
			employee = &employees[i]
			break
		}
	}
	if employee == nil {
		response.RespondWithError(w, domain.ErrNotFound)
		return
	}

	// Update bank details
	employee.BankName = req.BankName
	employee.BankCode = req.BankCode
	employee.BankAccountNumber = req.BankAccountNumber

	updated, err := h.employeeService.UpdateEmployee(r.Context(), employee)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, updated)
}

// GetPayslips handles GET /v1/me/payslips
func (h *SelfServiceHandler) GetPayslips(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	// Get all payroll runs for this business
	runs, err := h.payrollService.ListByBusinessID(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	// Filter entries for this employee's user
	type payslipSummary struct {
		RunID  uint   `json:"run_id"`
		Period string `json:"period"`
		Status string `json:"status"`
		NetPay int64  `json:"net_pay"`
	}

	var payslips []payslipSummary
	for _, run := range runs {
		if run.Status != domain.StatusCompleted && run.Status != domain.StatusProcessing {
			continue
		}
		for _, entry := range run.Entries {
			if entry.Employee != nil && entry.Employee.UserID != nil && *entry.Employee.UserID == claims.UserID {
				payslips = append(payslips, payslipSummary{
					RunID:  run.ID,
					Period: run.Period.Format("2006-01"),
					Status: string(run.Status),
					NetPay: entry.NetPay,
				})
			}
		}
	}

	response.RespondWithJSON(w, http.StatusOK, payslips)
}
