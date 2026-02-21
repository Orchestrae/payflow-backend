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

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type DeductionRuleHandler struct {
	deductionRuleService service.DeductionRuleService
	validate             *validator.Validate
}

func NewDeductionRuleHandler(svc service.DeductionRuleService) *DeductionRuleHandler {
	return &DeductionRuleHandler{
		deductionRuleService: svc,
		validate:             validator.New(),
	}
}

// CreateDeductionRule handles POST /deduction-rules
func (h *DeductionRuleHandler) CreateDeductionRule(w http.ResponseWriter, r *http.Request) {
	var req request.CreateDeductionRuleRequest
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

	rule := &domain.DeductionRule{
		BusinessID:       claims.BusinessID,
		Name:             req.Name,
		Type:             req.Type,
		Value:            req.Value,
		CalculationBasis: req.CalculationBasis,
	}

	createdRule, err := h.deductionRuleService.CreateDeductionRule(r.Context(), rule)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, createdRule)
}

// ListDeductionRules handles GET /deduction-rules
func (h *DeductionRuleHandler) ListDeductionRules(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	rules, err := h.deductionRuleService.ListByBusinessID(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, rules)
}

// UpdateDeductionRule handles PUT /deduction-rules/{ruleID}
func (h *DeductionRuleHandler) UpdateDeductionRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := strconv.ParseUint(chi.URLParam(r, "ruleID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	var req request.CreateDeductionRuleRequest
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

	rule := &domain.DeductionRule{
		BusinessID:       claims.BusinessID,
		Name:             req.Name,
		Type:             req.Type,
		Value:            req.Value,
		CalculationBasis: req.CalculationBasis,
	}
	rule.ID = uint(ruleID)

	updatedRule, err := h.deductionRuleService.UpdateDeductionRule(r.Context(), rule)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, updatedRule)
}

// DeleteDeductionRule handles DELETE /deduction-rules/{ruleID}
func (h *DeductionRuleHandler) DeleteDeductionRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := strconv.ParseUint(chi.URLParam(r, "ruleID"), 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	if err := h.deductionRuleService.DeleteDeductionRule(r.Context(), uint(ruleID), claims.BusinessID); err != nil {
		response.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
