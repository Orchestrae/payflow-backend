package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"payflow/internal/domain"
	"payflow/internal/platform/paystack"
	"payflow/internal/repository"
	"payflow/internal/service"
)

// PaystackWebhookHandler handles Paystack webhook events.
type PaystackWebhookHandler struct {
	secretKey    string
	transferRepo repository.TransferRepository
	walletSvc    service.WalletService
}

// NewPaystackWebhookHandler creates a new Paystack webhook handler.
func NewPaystackWebhookHandler(secretKey string, transferRepo repository.TransferRepository, walletSvc service.WalletService) *PaystackWebhookHandler {
	return &PaystackWebhookHandler{
		secretKey:    secretKey,
		transferRepo: transferRepo,
		walletSvc:    walletSvc,
	}
}

// HandleWebhook handles incoming Paystack webhook events.
// Verifies HMAC-SHA512 signature, then routes by event type.
func (h *PaystackWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read Paystack webhook body", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Verify HMAC-SHA512 signature — mandatory
	signature := r.Header.Get("x-paystack-signature")
	if signature == "" {
		slog.Error("Missing Paystack webhook signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !h.verifySignature(bodyBytes, signature) {
		slog.Error("Paystack webhook signature verification failed")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse webhook payload
	var payload paystack.WebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		slog.Error("Failed to parse Paystack webhook payload", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	slog.Info("Received Paystack webhook", "event", payload.Event)

	// Route by event type
	switch payload.Event {
	case "transfer.success":
		h.handleTransferStatus(r.Context(), payload.Data, "success")
	case "transfer.failed":
		h.handleTransferStatus(r.Context(), payload.Data, "failed")
	case "transfer.reversed":
		h.handleTransferStatus(r.Context(), payload.Data, "reversed")
	case "charge.success":
		h.handleChargeSuccess(r.Context(), payload.Data)
	default:
		slog.Debug("Unhandled Paystack event", "event", payload.Event)
	}

	// Always acknowledge
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// handleChargeSuccess processes Paystack deposit (charge.success) into the wallet.
func (h *PaystackWebhookHandler) handleChargeSuccess(ctx context.Context, data map[string]interface{}) {
	if h.walletSvc == nil {
		slog.Warn("Paystack charge.success received but wallet service not configured")
		return
	}

	reference, _ := data["reference"].(string)
	if reference == "" {
		slog.Warn("Paystack charge.success missing reference")
		return
	}

	// Extract amount (Paystack sends amount in kobo as number)
	var amount int64
	switch v := data["amount"].(type) {
	case float64:
		amount = int64(v)
	case json.Number:
		a, _ := v.Int64()
		amount = a
	}
	if amount <= 0 {
		slog.Warn("Paystack charge.success invalid amount", "reference", reference)
		return
	}

	// Extract currency
	currency, _ := data["currency"].(string)
	if currency == "" {
		currency = "NGN"
	}

	now := time.Now()
	notification := &domain.DepositNotification{
		Reference:   reference,
		Amount:      amount,
		Currency:    currency,
		Description: fmt.Sprintf("Paystack deposit: %s", reference),
		Status:      "completed",
		ProcessedAt: now,
	}

	// Try to extract business_id from metadata (set by HandleInitiateDeposit)
	var businessID uint
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		switch v := metadata["business_id"].(type) {
		case float64:
			businessID = uint(v)
		case json.Number:
			bid, _ := v.Int64()
			businessID = uint(bid)
		}
	}

	if businessID > 0 {
		// Direct deposit via Paystack checkout — we know the business
		if err := h.walletSvc.RecordDeposit(ctx, businessID, notification); err != nil {
			slog.Error("Failed to record Paystack deposit", "reference", reference, "business_id", businessID, "error", err)
		} else {
			slog.Info("Paystack deposit recorded", "reference", reference, "amount", amount, "business_id", businessID)
		}
		return
	}

	// Fallback: try account_reference for KoraPay virtual account deposits
	accountRef := ""
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		accountRef, _ = metadata["account_reference"].(string)
	}
	if accountRef == "" {
		if dedicatedAccount, ok := data["dedicated_account"].(map[string]interface{}); ok {
			if acct, ok := dedicatedAccount["account_number"].(string); ok {
				accountRef = acct
			}
		}
	}

	if accountRef == "" {
		slog.Warn("Paystack charge.success missing both business_id and account reference", "reference", reference)
		return
	}

	if err := h.walletSvc.RecordDepositByAccountReference(ctx, accountRef, notification); err != nil {
		slog.Error("Failed to record Paystack deposit by account ref", "reference", reference, "error", err)
	} else {
		slog.Info("Paystack deposit recorded via account ref", "reference", reference, "amount", amount, "account_ref", accountRef)
	}
}

// handleTransferStatus updates transfer status in the database based on webhook event.
func (h *PaystackWebhookHandler) handleTransferStatus(ctx context.Context, data map[string]interface{}, status string) {
	reference, _ := data["reference"].(string)
	if reference == "" {
		slog.Warn("Paystack transfer webhook missing reference")
		return
	}

	transfer, err := h.transferRepo.FindByReference(ctx, reference)
	if err != nil {
		slog.Warn("Transfer not found for Paystack webhook", "reference", reference, "error", err)
		return
	}

	transfer.Status = status
	transfer.ProviderStatus = status
	if msg, ok := data["reason"].(string); ok {
		transfer.ProviderMessage = msg
	}
	if transferCode, ok := data["transfer_code"].(string); ok {
		transfer.TransactionID = transferCode
	}

	if err := h.transferRepo.Update(ctx, transfer); err != nil {
		slog.Error("Failed to update transfer from Paystack webhook", "reference", reference, "error", err)
	} else {
		slog.Info("Transfer status updated from Paystack webhook", "reference", reference, "status", status)
	}
}

// verifySignature verifies Paystack HMAC-SHA512 signature.
func (h *PaystackWebhookHandler) verifySignature(body []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(h.secretKey))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}
