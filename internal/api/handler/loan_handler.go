package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
)

// LoanHandler handles employee loan endpoints.
type LoanHandler struct {
	loanService service.LoanService
	validate    *validator.Validate
}

// NewLoanHandler creates a new loan handler.
func NewLoanHandler(svc service.LoanService) *LoanHandler {
	return &LoanHandler{loanService: svc, validate: validator.New()}
}

type createLoanRequest struct {
	EmployeeID       uint   `json:"employee_id" validate:"required"`
	LoanAmount       int64  `json:"loan_amount" validate:"required,min=1"`
	MonthlyDeduction int64  `json:"monthly_deduction" validate:"required,min=1"`
	StartDate        string `json:"start_date" validate:"required"`
	Description      string `json:"description"`
}

// CreateLoan handles POST /v1/loans
func (h *LoanHandler) CreateLoan(w http.ResponseWriter, r *http.Request) {
	var req createLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	loan := &domain.EmployeeLoan{
		BusinessID:       claims.BusinessID,
		EmployeeID:       req.EmployeeID,
		LoanAmount:       req.LoanAmount,
		MonthlyDeduction: req.MonthlyDeduction,
		StartDate:        startDate,
		Description:      req.Description,
	}

	created, err := h.loanService.Create(r.Context(), loan)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, created)
}

// ListLoans handles GET /v1/loans
func (h *LoanHandler) ListLoans(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	loans, total, err := h.loanService.ListByBusiness(r.Context(), claims.BusinessID, page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data":  loans,
		"total": total,
	})
}

// CancelLoan handles PATCH /v1/loans/{id}/cancel
func (h *LoanHandler) CancelLoan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	if err := h.loanService.Cancel(r.Context(), uint(id), claims.BusinessID); err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "loan cancelled"})
}
