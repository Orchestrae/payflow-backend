// internal/api/handler/employee_handler.go
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"payflow/internal/api/middleware"
	"payflow/internal/api/request"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type EmployeeHandler struct {
	employeeService service.EmployeeService
	validate        *validator.Validate
}

func NewEmployeeHandler(svc service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{
		employeeService: svc,
		validate:        validator.New(),
	}
}

// CreateEmployee handles POST /employees
func (h *EmployeeHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req request.CreateEmployeeRequest
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
		response.RespondWithError(w, domain.ErrInternalServer) // Should not happen
		return
	}

	employee := &domain.Employee{
		BusinessID:        claims.BusinessID,
		CadreID:           req.CadreID,
		FullName:          req.FullName,
		Email:             req.Email,
		BankName:          req.BankName,
		BankCode:          req.BankCode,
		BankAccountNumber: req.BankAccountNumber,
		IsActive:          true, // New employees are active by default
	}

	createdEmployee, err := h.employeeService.CreateEmployee(r.Context(), employee)
	if err != nil {
		slog.Error("Failed to create employee", "error", err)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, createdEmployee)
}

// GetEmployeeByID handles GET /employees/{employeeID}
func (h *EmployeeHandler) GetEmployeeByID(w http.ResponseWriter, r *http.Request) {
	employeeID, err := strconv.ParseUint(chi.URLParam(r, "employeeID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	employee, err := h.employeeService.GetByID(r.Context(), uint(employeeID), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, employee)
}

// ListEmployees handles GET /employees
func (h *EmployeeHandler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	employees, err := h.employeeService.ListByBusinessID(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, employees)
}

// UpdateEmployee handles PUT /employees/{employeeID}
func (h *EmployeeHandler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	employeeID, err := strconv.ParseUint(chi.URLParam(r, "employeeID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	var req request.UpdateEmployeeRequest
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
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	employee := &domain.Employee{
		Model:             domain.Model{ID: uint(employeeID)},
		BusinessID:        claims.BusinessID,
		CadreID:           req.CadreID,
		FullName:          req.FullName,
		Email:             req.Email,
		BankName:          req.BankName,
		BankCode:          req.BankCode,
		BankAccountNumber: req.BankAccountNumber,
	}

	updatedEmployee, err := h.employeeService.UpdateEmployee(r.Context(), employee)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, updatedEmployee)
}

// DeactivateEmployee handles PATCH /employees/{employeeID}/deactivate
func (h *EmployeeHandler) DeactivateEmployee(w http.ResponseWriter, r *http.Request) {
	employeeID, err := strconv.ParseUint(chi.URLParam(r, "employeeID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrInternalServer)
		return
	}

	if err := h.employeeService.DeactivateEmployee(r.Context(), uint(employeeID), claims.BusinessID); err != nil {
		response.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
