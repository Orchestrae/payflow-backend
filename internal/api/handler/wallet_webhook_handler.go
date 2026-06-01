// internal/api/handler/wallet_webhook_handler.go
// Korapay deposit webhook handling, extracted from wallet_handler.go
package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"payflow/internal/domain"
)

func (h *WalletHandler) HandleDepositWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Verify webhook signature (HMAC-SHA256) — mandatory
	signature := r.Header.Get("x-korapay-signature")
	if signature == "" {
		slog.Error("Missing webhook signature header")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !h.verifyWebhookSignature(bodyBytes, signature) {
		slog.Error("Webhook signature verification failed")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the full webhook payload
	var webhookPayload struct {
		Event string                 `json:"event"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &webhookPayload); err != nil {
		slog.Error("Failed to parse webhook payload", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	slog.Debug("Received deposit webhook", "event", webhookPayload.Event)

	// Only process charge.success events
	if webhookPayload.Event != "charge.success" && webhookPayload.Event != "" {
		// Acknowledge non-charge events gracefully
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"acknowledged"}`))
		return
	}

	data := webhookPayload.Data
	if data == nil {
		slog.Error("Missing data in webhook payload")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Extract account reference from nested structure:
	// data.virtual_bank_account_details.virtual_bank_account.account_reference
	accountRef := ""
	accountNumber := ""
	if vbaDetails, ok := data["virtual_bank_account_details"].(map[string]interface{}); ok {
		if vba, ok := vbaDetails["virtual_bank_account"].(map[string]interface{}); ok {
			accountRef = parseString(vba["account_reference"])
			accountNumber = parseString(vba["account_number"])
		}
	}

	if accountRef == "" {
		slog.Error("Missing account_reference in webhook payload")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	notification := h.parseKorapayDepositNotification(data, accountRef, accountNumber)
	if notification == nil {
		slog.Error("Failed to parse deposit notification from webhook")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := h.walletService.RecordDepositByAccountReference(r.Context(), accountRef, notification); err != nil {
		slog.Error("Failed to record deposit from webhook", "error", err, "account_reference", accountRef)
	}

	// Always return 200 OK to acknowledge receipt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// verifyWebhookSignature verifies the HMAC-SHA256 signature of a Korapay webhook
func (h *WalletHandler) verifyWebhookSignature(body []byte, signature string) bool {
	// Korapay signs only the "data" object
	var payload struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.koraSecretKey))
	mac.Write(payload.Data)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

// parseKorapayDepositNotification parses the data object from a Korapay charge.success webhook
func (h *WalletHandler) parseKorapayDepositNotification(data map[string]interface{}, accountRef, accountNumber string) *domain.DepositNotification {
	reference := parseString(data["reference"])
	if reference == "" {
		return nil
	}

	amountFloat, ok := data["amount"].(float64)
	if !ok {
		if amountStr, ok := data["amount"].(string); ok {
			if parsed, err := strconv.ParseFloat(amountStr, 64); err == nil {
				amountFloat = parsed
			}
		}
	}

	// Convert to kobo safely using math.Round to avoid float precision issues
	amount := int64(math.Round(amountFloat * 100))

	status := parseString(data["status"])
	if status == "" {
		status = "success"
	}

	currency := parseString(data["currency"])
	if currency == "" {
		currency = "NGN"
	}

	description := parseString(data["narration"])

	processedAt := time.Now()
	if timestampStr := parseString(data["created_at"]); timestampStr != "" {
		if parsed, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			processedAt = parsed
		}
	}

	// Extract payer bank account from nested structure
	var payerBankAccount *domain.PayerBankAccount
	if vbaDetails, ok := data["virtual_bank_account_details"].(map[string]interface{}); ok {
		if payerData, ok := vbaDetails["payer_bank_account"].(map[string]interface{}); ok {
			payerBankAccount = &domain.PayerBankAccount{
				AccountNumber: parseString(payerData["account_number"]),
				AccountName:   parseString(payerData["account_name"]),
				BankName:      parseString(payerData["bank_name"]),
			}
		}
	}

	return &domain.DepositNotification{
		Provider:         domain.ProviderKorapay,
		Reference:        reference,
		AccountReference: accountRef,
		AccountNumber:    accountNumber,
		Amount:           amount,
		Currency:         currency,
		Status:           status,
		Description:      description,
		ProcessedAt:      processedAt,
		PayerBankAccount: payerBankAccount,
	}
}

// HandleSandboxCredit handles POST /v1/wallets/sandbox/credit
