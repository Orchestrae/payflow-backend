package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/internal/service"
	"payflow/internal/service/report"
)

// ReportHandler handles report download endpoints.
type ReportHandler struct {
	payrollSvc   service.PayrollService
	businessRepo repository.BusinessRepository
}

// NewReportHandler creates a new report handler.
func NewReportHandler(payrollSvc service.PayrollService, businessRepo repository.BusinessRepository) *ReportHandler {
	return &ReportHandler{
		payrollSvc:   payrollSvc,
		businessRepo: businessRepo,
	}
}

func (h *ReportHandler) loadPayrollAndBusiness(w http.ResponseWriter, r *http.Request) (*domain.PayrollRun, *domain.Business, bool) {
	runID, err := strconv.ParseUint(chi.URLParam(r, "runID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return nil, nil, false
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return nil, nil, false
	}

	run, err := h.payrollSvc.GetByID(r.Context(), uint(runID), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return nil, nil, false
	}

	business, err := h.businessRepo.FindByID(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return nil, nil, false
	}

	return run, business, true
}

func (h *ReportHandler) serveCSV(w http.ResponseWriter, data []byte, filename string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// HandlePAYEReport handles GET /v1/payroll-runs/{runID}/reports/paye
func (h *ReportHandler) HandlePAYEReport(w http.ResponseWriter, r *http.Request) {
	run, business, ok := h.loadPayrollAndBusiness(w, r)
	if !ok {
		return
	}

	data, err := report.GeneratePAYEReport(run, business)
	if err != nil {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	h.serveCSV(w, data, fmt.Sprintf("paye_return_%s.csv", run.Period.Format("2006-01")))
}

// HandlePensionSchedule handles GET /v1/payroll-runs/{runID}/reports/pension
func (h *ReportHandler) HandlePensionSchedule(w http.ResponseWriter, r *http.Request) {
	run, business, ok := h.loadPayrollAndBusiness(w, r)
	if !ok {
		return
	}

	data, err := report.GeneratePensionSchedule(run, business)
	if err != nil {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	h.serveCSV(w, data, fmt.Sprintf("pension_schedule_%s.csv", run.Period.Format("2006-01")))
}

// HandleNHFSchedule handles GET /v1/payroll-runs/{runID}/reports/nhf
func (h *ReportHandler) HandleNHFSchedule(w http.ResponseWriter, r *http.Request) {
	run, business, ok := h.loadPayrollAndBusiness(w, r)
	if !ok {
		return
	}

	data, err := report.GenerateNHFSchedule(run, business)
	if err != nil {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	h.serveCSV(w, data, fmt.Sprintf("nhf_schedule_%s.csv", run.Period.Format("2006-01")))
}

// HandleBankSchedule handles GET /v1/payroll-runs/{runID}/reports/bank-schedule
func (h *ReportHandler) HandleBankSchedule(w http.ResponseWriter, r *http.Request) {
	run, business, ok := h.loadPayrollAndBusiness(w, r)
	if !ok {
		return
	}

	data, err := report.GenerateBankSchedule(run, business)
	if err != nil {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	h.serveCSV(w, data, fmt.Sprintf("bank_schedule_%s.csv", run.Period.Format("2006-01")))
}

// HandlePayslip handles GET /v1/payroll-runs/{runID}/payslips/{employeeID}
func (h *ReportHandler) HandlePayslip(w http.ResponseWriter, r *http.Request) {
	run, business, ok := h.loadPayrollAndBusiness(w, r)
	if !ok {
		return
	}

	employeeID, err := strconv.ParseUint(chi.URLParam(r, "employeeID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Find the entry for this employee
	var entry *domain.PayrollRunEntry
	for i := range run.Entries {
		if run.Entries[i].EmployeeID == uint(employeeID) {
			entry = &run.Entries[i]
			break
		}
	}
	if entry == nil {
		response.RespondWithError(w, domain.ErrNotFound)
		return
	}

	data, err := report.GeneratePayslip(entry, business, run.Period)
	if err != nil {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	employeeName := "employee"
	if entry.Employee != nil {
		employeeName = sanitizeForFilename(entry.Employee.FullName)
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="payslip_%s_%s.pdf"`, employeeName, run.Period.Format("2006-01")))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// HandleAllPayslips handles GET /v1/payroll-runs/{runID}/payslips
func (h *ReportHandler) HandleAllPayslips(w http.ResponseWriter, r *http.Request) {
	run, business, ok := h.loadPayrollAndBusiness(w, r)
	if !ok {
		return
	}

	data, err := report.GenerateAllPayslips(run, business)
	if err != nil {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="payslips_%s.zip"`, run.Period.Format("2006-01")))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// HandlePayrollSummary handles GET /v1/payroll-runs/{runID}/reports/summary
func (h *ReportHandler) HandlePayrollSummary(w http.ResponseWriter, r *http.Request) {
	run, business, ok := h.loadPayrollAndBusiness(w, r)
	if !ok {
		return
	}

	data, err := report.GeneratePayrollSummary(run, business)
	if err != nil {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	h.serveCSV(w, data, fmt.Sprintf("payroll_summary_%s.csv", run.Period.Format("2006-01")))
}

func sanitizeForFilename(name string) string {
	var result []byte
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, byte(c))
		} else if c == ' ' {
			result = append(result, '_')
		}
	}
	return string(result)
}
