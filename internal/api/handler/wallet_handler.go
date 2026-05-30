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
	"strings"
	"time"

	"fmt"

	"payflow/internal/api/middleware"
	"payflow/internal/api/request"
	"payflow/internal/api/response"
	"payflow/internal/config"
	"payflow/internal/domain"
	"payflow/internal/platform/billing"
	"payflow/internal/platform/korapay"
	"payflow/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type WalletHandler struct {
	walletService        service.WalletService
	accountHolderService service.AccountHolderService
	koraClient           *korapay.Client
	koraBaseURL          string
	koraSecretKey        string
	paystackSecretKey    string
	paystackBaseURL      string
	appURL               string
	validate             *validator.Validate
}

func NewWalletHandler(
	walletService service.WalletService,
	accountHolderService service.AccountHolderService,
	cfg *config.Config,
	koraClient *korapay.Client,
) *WalletHandler {
	return &WalletHandler{
		walletService:        walletService,
		accountHolderService: accountHolderService,
		koraClient:           koraClient,
		koraBaseURL:          cfg.KoraPayBaseURL,
		koraSecretKey:        cfg.KoraPayAPIKey,
		paystackSecretKey:    cfg.PaystackSecretKey,
		paystackBaseURL:      cfg.PaystackBaseURL,
		appURL:               cfg.AppURL,
		validate:             validator.New(),
	}
}

// HandleCreateVirtualAccount handles POST /v1/wallets/virtual-account
func (h *WalletHandler) HandleCreateVirtualAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}
	businessID := claims.BusinessID

	var req request.CreateVirtualAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode create virtual account request", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		slog.Error("Create virtual account validation failed", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	domainReq := &domain.CreateVirtualAccountRequest{
		BusinessID:       businessID,
		AccountName:      req.AccountName,
		AccountReference: req.AccountReference,
		CustomerName:     req.CustomerName,
		CustomerEmail:    req.CustomerEmail,
		BVN:              req.BVN,
		NIN:              req.NIN,
		BankCode:         req.BankCode,
		Permanent:        req.Permanent,
	}

	result, err := h.walletService.CreateVirtualAccount(r.Context(), businessID, domainReq)
	if err != nil {
		slog.Error("Failed to create virtual account", "error", err, "business_id", businessID)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, result)
}

// HandleGetWallet handles GET /v1/wallets
func (h *WalletHandler) HandleGetWallet(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	wallet, err := h.walletService.GetWallet(r.Context(), claims.BusinessID)
	if err != nil {
		slog.Error("Failed to get wallet", "error", err, "business_id", claims.BusinessID)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, wallet)
}

// HandleGetBalance handles GET /v1/wallets/balance
func (h *WalletHandler) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	balance, err := h.walletService.GetBalance(r.Context(), claims.BusinessID)
	if err != nil {
		slog.Error("Failed to get balance", "error", err, "business_id", claims.BusinessID)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"balance":  balance,
		"currency": "NGN",
	})
}

// HandleGetTransactions handles GET /v1/wallets/transactions
func (h *WalletHandler) HandleGetTransactions(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	transactions, total, err := h.walletService.GetTransactions(r.Context(), claims.BusinessID, page, limit)
	if err != nil {
		slog.Error("Failed to get transactions", "error", err, "business_id", claims.BusinessID)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"total":        total,
		"page":         page,
		"limit":        limit,
	})
}

// HandleDepositWebhook handles POST /korapay/webhooks/deposit
// Receives deposit notifications from KoraPay virtual account
// This is a public endpoint that KoraPay will call
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
func (h *WalletHandler) HandleSandboxCredit(w http.ResponseWriter, r *http.Request) {
	if !h.isSandboxMode() {
		slog.Warn("Sandbox credit attempted in non-sandbox environment")
		response.RespondWithError(w, domain.ErrForbidden)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}
	businessID := claims.BusinessID

	var req request.SandboxCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode sandbox credit request", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		slog.Error("Sandbox credit validation failed", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if req.Currency == "" {
		req.Currency = "NGN"
	}

	wallet, err := h.walletService.GetWallet(r.Context(), businessID)
	if err != nil {
		slog.Error("Wallet not found for sandbox credit", "error", err, "business_id", businessID)
		response.RespondWithError(w, err)
		return
	}

	if wallet.VirtualAccountNumber != req.AccountNumber {
		slog.Error("Account number mismatch for sandbox credit",
			"provided", req.AccountNumber,
			"wallet", wallet.VirtualAccountNumber,
			"business_id", businessID)
		response.RespondWithError(w, domain.ErrForbidden)
		return
	}

	sandboxReq := korapay.VirtualAccountSandboxCreditRequest{
		AccountNumber: req.AccountNumber,
		Amount:        req.Amount,
		Currency:      req.Currency,
	}

	koraResponse, err := h.koraClient.SandboxCreditVirtualAccount(r.Context(), sandboxReq)
	if err != nil {
		slog.Error("KoraPay sandbox credit failed", "error", err, "account_number", req.AccountNumber)
		response.RespondWithError(w, err)
		return
	}

	if !koraResponse.Status {
		slog.Error("KoraPay sandbox credit returned error", "message", koraResponse.Message)
		response.RespondWithError(w, domain.ErrPaymentGatewayFailed)
		return
	}

	// Convert amount to kobo safely
	amountInKobo := int64(req.Amount) * 100

	notification := &domain.DepositNotification{
		Provider:         domain.ProviderKorapay,
		Reference:        "SANDBOX-CREDIT-" + strconv.FormatInt(time.Now().Unix(), 10),
		AccountNumber:    req.AccountNumber,
		AccountReference: wallet.VirtualAccountReference,
		Amount:           amountInKobo,
		Currency:         req.Currency,
		Status:           "success",
		Description:      "Sandbox test credit",
		ProcessedAt:      time.Now(),
	}

	if err := h.walletService.RecordDeposit(r.Context(), businessID, notification); err != nil {
		slog.Error("Failed to record sandbox deposit", "error", err, "business_id", businessID)
		response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"status":  true,
			"message": "Virtual bank account credited successfully (deposit recording may have failed)",
		})
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  true,
		"message": "Virtual bank account credited successfully",
	})
}

// isSandboxMode checks if we're in sandbox/test mode
// Korapay uses the same base URL for sandbox and live; the API key determines the environment.
// Test keys start with "sk_test_" while live keys start with "sk_live_"
func (h *WalletHandler) isSandboxMode() bool {
	if strings.HasPrefix(h.koraSecretKey, "sk_test_") || strings.HasPrefix(h.koraSecretKey, "pk_test_") {
		return true
	}
	return strings.Contains(strings.ToLower(h.koraBaseURL), "sandbox") ||
		strings.Contains(strings.ToLower(h.koraBaseURL), "test") ||
		strings.Contains(strings.ToLower(h.koraBaseURL), "staging")
}

// parseString safely extracts a string from an interface{}
func parseString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// ============================================================================
// Account Holder / KYC Handlers
// ============================================================================

// HandleCreateAccountHolder handles POST /v1/wallets/account-holders
func (h *WalletHandler) HandleCreateAccountHolder(w http.ResponseWriter, r *http.Request) {
	var req request.CreateAccountHolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode create account holder request", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		slog.Error("Create account holder validation failed", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	domainReq := &domain.CreateAccountHolderRequest{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		UseCase:        req.UseCase,
		Type:           req.Type,
		DateOfBirth:    req.DateOfBirth,
		Nationality:    req.Nationality,
		Occupation:     req.Occupation,
		Email:          req.Email,
		Phone:          req.Phone,
		BankIDNumber:   req.BankIDNumber,
		SourceOfInflow: req.SourceOfInflow,
		Metadata:       req.Metadata,
	}

	if req.SourceOfInflowDocument != nil {
		domainReq.SourceOfInflowDocument = &domain.FileReference{
			Reference: req.SourceOfInflowDocument.Reference,
		}
	}

	if req.Selfie != nil {
		domainReq.Selfie = &domain.FileReference{
			Reference: req.Selfie.Reference,
		}
	}

	if req.Identification != nil {
		domainReq.Identification = &domain.AccountHolderIdentification{
			Type:       req.Identification.Type,
			Number:     req.Identification.Number,
			IssuedDate: req.Identification.IssuedDate,
			ExpiryDate: req.Identification.ExpiryDate,
			Country:    req.Identification.Country,
		}
		if req.Identification.DocumentFront != nil {
			domainReq.Identification.DocumentFront = &domain.FileReference{
				Reference: req.Identification.DocumentFront.Reference,
			}
		}
		if req.Identification.DocumentBack != nil {
			domainReq.Identification.DocumentBack = &domain.FileReference{
				Reference: req.Identification.DocumentBack.Reference,
			}
		}
	}

	if req.ProofOfAddress != nil {
		domainReq.ProofOfAddress = &domain.AccountHolderProofOfAddress{
			Type: req.ProofOfAddress.Type,
		}
		if req.ProofOfAddress.Document != nil {
			domainReq.ProofOfAddress.Document = &domain.FileReference{
				Reference: req.ProofOfAddress.Document.Reference,
			}
		}
	}

	if req.Address != nil {
		domainReq.Address = &domain.AccountHolderAddress{
			Country: req.Address.Country,
			Zip:     req.Address.Zip,
			Address: req.Address.Address,
			State:   req.Address.State,
			City:    req.Address.City,
		}
	}

	if req.Employment != nil {
		domainReq.Employment = &domain.AccountHolderEmployment{
			Status:      req.Employment.Status,
			Employer:    req.Employment.Employer,
			Description: req.Employment.Description,
		}
	}

	result, err := h.accountHolderService.CreateAccountHolder(r.Context(), domainReq)
	if err != nil {
		slog.Error("Failed to create account holder", "error", err)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, result)
}

// HandleGetAccountHolderDetails handles GET /v1/wallets/account-holders/{reference}/details
func (h *WalletHandler) HandleGetAccountHolderDetails(w http.ResponseWriter, r *http.Request) {
	reference := chi.URLParam(r, "reference")
	if reference == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	details, err := h.accountHolderService.GetAccountHolderDetails(r.Context(), reference)
	if err != nil {
		slog.Error("Failed to get account holder details", "error", err, "reference", reference)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, details)
}

// HandleUpdateAccountHolderKYC handles PATCH /v1/wallets/account-holders/{reference}/update-kyc
func (h *WalletHandler) HandleUpdateAccountHolderKYC(w http.ResponseWriter, r *http.Request) {
	reference := chi.URLParam(r, "reference")
	if reference == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	var req request.UpdateAccountHolderKYCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode update account holder KYC request", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		slog.Error("Update account holder KYC validation failed", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	domainReq := &domain.UpdateAccountHolderKYCRequest{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		SourceOfInflow: req.SourceOfInflow,
	}

	if req.SourceOfInflowDocument != nil {
		domainReq.SourceOfInflowDocument = &domain.FileReference{
			Reference: req.SourceOfInflowDocument.Reference,
		}
	}

	if req.Selfie != nil {
		domainReq.Selfie = &domain.FileReference{
			Reference: req.Selfie.Reference,
		}
	}

	if req.Identification != nil {
		domainReq.Identification = &domain.AccountHolderIdentification{
			Type:       req.Identification.Type,
			Number:     req.Identification.Number,
			IssuedDate: req.Identification.IssuedDate,
			ExpiryDate: req.Identification.ExpiryDate,
			Country:    req.Identification.Country,
		}
		if req.Identification.DocumentFront != nil {
			domainReq.Identification.DocumentFront = &domain.FileReference{
				Reference: req.Identification.DocumentFront.Reference,
			}
		}
		if req.Identification.DocumentBack != nil {
			domainReq.Identification.DocumentBack = &domain.FileReference{
				Reference: req.Identification.DocumentBack.Reference,
			}
		}
	}

	if req.ProofOfAddress != nil {
		domainReq.ProofOfAddress = &domain.AccountHolderProofOfAddress{
			Type: req.ProofOfAddress.Type,
		}
		if req.ProofOfAddress.Document != nil {
			domainReq.ProofOfAddress.Document = &domain.FileReference{
				Reference: req.ProofOfAddress.Document.Reference,
			}
		}
	}

	result, err := h.accountHolderService.UpdateAccountHolderKYC(r.Context(), reference, domainReq)
	if err != nil {
		slog.Error("Failed to update account holder KYC", "error", err, "reference", reference)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, result)
}

// HandleGenerateFileUploadURL handles POST /v1/wallets/files/generate-upload-url
func (h *WalletHandler) HandleGenerateFileUploadURL(w http.ResponseWriter, r *http.Request) {
	var req request.GenerateFileUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode generate file upload URL request", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		slog.Error("Generate file upload URL validation failed", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	domainReq := &domain.GenerateFileUploadURLRequest{
		Reference:   req.Reference,
		Purpose:     req.Purpose,
		ContentType: req.ContentType,
	}

	result, err := h.accountHolderService.GenerateFileUploadURL(r.Context(), domainReq)
	if err != nil {
		slog.Error("Failed to generate file upload URL", "error", err)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, result)
}

// HandleInitiateDeposit handles POST /v1/wallets/deposit
// Initializes a Paystack payment (card/transfer/USSD) and returns the payment URL.
func (h *WalletHandler) HandleInitiateDeposit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount int64  `json:"amount" validate:"required,min=10000"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}

	if h.paystackSecretKey == "" {
		response.RespondWithJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Payment provider not configured",
		})
		return
	}

	ref := fmt.Sprintf("DEP-%d-%d", claims.BusinessID, time.Now().UnixNano())
	email := req.Email
	if email == "" {
		email = fmt.Sprintf("biz-%d@payflow.local", claims.BusinessID)
	}
	callbackURL := fmt.Sprintf("%s/wallet?deposit=success", h.appURL)

	// Initialize Paystack transaction via billing client
	billingClient := billing.NewPaystackBillingClient(h.paystackSecretKey, h.paystackBaseURL)
	paymentURL, err := billingClient.InitializeTransaction(r.Context(), email, req.Amount, ref, "", callbackURL)
	if err != nil {
		slog.Error("Failed to initialize deposit", "error", err)
		response.RespondWithError(w, domain.ErrPaymentGatewayFailed)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"payment_url": paymentURL,
		"reference":   ref,
		"amount":      req.Amount,
		"message":     "Redirect to complete payment. Wallet will be credited on success.",
	})
}
