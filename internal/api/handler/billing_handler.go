package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"payflow/internal/api/middleware"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service/platform"
)

// BillingHandler handles subscription and billing endpoints.
type BillingHandler struct {
	billingService platform.BillingService
}

// NewBillingHandler creates a new billing handler.
func NewBillingHandler(svc platform.BillingService) *BillingHandler {
	return &BillingHandler{billingService: svc}
}

// GetPlans handles GET /v1/billing/plans
func (h *BillingHandler) GetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.billingService.GetPlans(r.Context())
	if err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusOK, plans)
}

// GetSubscription handles GET /v1/billing/subscription
func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	sub, err := h.billingService.GetSubscription(r.Context(), claims.BusinessID)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}
	response.RespondWithJSON(w, http.StatusOK, sub)
}

type subscribeRequest struct {
	Tier        string `json:"tier" validate:"required"`
	CallbackURL string `json:"callback_url,omitempty"`
}

// Subscribe handles POST /v1/billing/subscribe
func (h *BillingHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	// Get admin email for Paystack customer
	paymentURL, err := h.billingService.Subscribe(r.Context(), claims.BusinessID, domain.PlanTier(req.Tier), "", req.CallbackURL)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	if paymentURL == "" {
		response.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Free plan activated",
		})
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]string{
		"payment_url": paymentURL,
		"message":     "Redirect to complete payment",
	})
}

// CancelSubscription handles POST /v1/billing/cancel
func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	if err := h.billingService.CancelSubscription(r.Context(), claims.BusinessID); err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Subscription cancelled. Downgraded to Free plan.",
	})
}

// ListInvoices handles GET /v1/billing/invoices
func (h *BillingHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	invoices, total, err := h.billingService.ListInvoices(r.Context(), claims.BusinessID, page, limit)
	if err != nil {
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data":  invoices,
		"total": total,
	})
}
