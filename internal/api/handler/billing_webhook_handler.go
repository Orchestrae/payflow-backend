package handler

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"

	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/internal/service/platform"
)

// BillingWebhookHandler handles Paystack billing/subscription webhook events.
type BillingWebhookHandler struct {
	paystackSecretKey string
	billingSvc        platform.BillingService
}

// NewBillingWebhookHandler creates a new billing webhook handler.
func NewBillingWebhookHandler(
	paystackSecretKey string,
	billingSvc platform.BillingService,
) *BillingWebhookHandler {
	return &BillingWebhookHandler{
		paystackSecretKey: paystackSecretKey,
		billingSvc:        billingSvc,
	}
}

// HandleWebhook handles POST /paystack/webhooks/billing
func (h *BillingWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Verify HMAC-SHA512 signature
	signature := r.Header.Get("X-Paystack-Signature")
	if signature == "" || !verifyPaystackBillingSignature(body, signature, h.paystackSecretKey) {
		log.Warn().Msg("Invalid Paystack billing webhook signature")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Parse event
	var event struct {
		Event string                 `json:"event"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	log.Info().Str("event", event.Event).Msg("Billing webhook received")

	switch event.Event {
	case "charge.success":
		h.handleChargeSuccess(r, event.Data)
	case "subscription.disable":
		h.handleSubscriptionDisable(r, event.Data)
	case "invoice.payment_failed":
		h.handleInvoicePaymentFailed(r, event.Data)
	default:
		log.Info().Str("event", event.Event).Msg("Unhandled billing webhook event")
	}

	// Always return 200 to acknowledge receipt
	w.WriteHeader(http.StatusOK)
}

func (h *BillingWebhookHandler) handleChargeSuccess(_ *http.Request, data map[string]interface{}) {
	metadata, _ := data["metadata"].(map[string]interface{})
	if metadata == nil {
		return
	}

	businessIDFloat, ok := metadata["business_id"].(float64)
	if !ok {
		return
	}
	businessID := uint(businessIDFloat)
	amountFloat, _ := data["amount"].(float64)
	reference, _ := data["reference"].(string)

	log.Info().Uint("business_id", businessID).Int64("amount", int64(amountFloat)).Str("reference", reference).Msg("Billing charge successful — subscription renewed")
}

func (h *BillingWebhookHandler) handleSubscriptionDisable(_ *http.Request, data map[string]interface{}) {
	customerData, _ := data["customer"].(map[string]interface{})
	email := ""
	if customerData != nil {
		email, _ = customerData["email"].(string)
	}
	log.Warn().Str("email", email).Msg("Subscription disabled via Paystack")
}

func (h *BillingWebhookHandler) handleInvoicePaymentFailed(_ *http.Request, data map[string]interface{}) {
	customer, _ := data["customer"].(map[string]interface{})
	email := ""
	if customer != nil {
		email, _ = customer["email"].(string)
	}
	log.Warn().Str("email", email).Msg("Invoice payment failed — subscription at risk")
}

func verifyPaystackBillingSignature(body []byte, signature, secretKey string) bool {
	mac := hmac.New(sha512.New, []byte(secretKey))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}
