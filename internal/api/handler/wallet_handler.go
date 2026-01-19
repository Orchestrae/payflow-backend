package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"payflow/internal/api/request"
	"payflow/internal/api/response"
	"payflow/internal/config"
	"payflow/internal/domain"
	"payflow/internal/platform/korapay"
	"payflow/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type WalletHandler struct {
	walletService        service.WalletService
	accountHolderService service.AccountHolderService
	koraClient           *korapay.Client // For sandbox credit only
	koraBaseURL          string          // To check if in sandbox
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
		validate:             validator.New(),
	}
}

// HandleCreateVirtualAccount handles POST /v1/wallets/virtual-account
// Creates a virtual account for a business
func (h *WalletHandler) HandleCreateVirtualAccount(w http.ResponseWriter, r *http.Request) {
	businessID := r.Context().Value("business_id").(uint)

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

	// Convert request to domain model
	domainReq := &domain.CreateVirtualAccountRequest{
		BusinessID:      businessID,
		AccountName:     req.AccountName,
		AccountReference: req.AccountReference,
		CustomerName:    req.CustomerName,
		CustomerEmail:   req.CustomerEmail,
		BVN:            req.BVN,
		NIN:            req.NIN,
		BankCode:       req.BankCode,
		Permanent:      req.Permanent,
	}

	// Create virtual account
	result, err := h.walletService.CreateVirtualAccount(r.Context(), businessID, domainReq)
	if err != nil {
		slog.Error("Failed to create virtual account", "error", err, "business_id", businessID)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, result)
}

// HandleGetWallet handles GET /v1/wallets
// Gets wallet details for a business
func (h *WalletHandler) HandleGetWallet(w http.ResponseWriter, r *http.Request) {
	businessID := r.Context().Value("business_id").(uint)

	wallet, err := h.walletService.GetWallet(r.Context(), businessID)
	if err != nil {
		slog.Error("Failed to get wallet", "error", err, "business_id", businessID)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, wallet)
}

// HandleGetBalance handles GET /v1/wallets/balance
// Gets current balance for a business wallet
func (h *WalletHandler) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	businessID := r.Context().Value("business_id").(uint)

	balance, err := h.walletService.GetBalance(r.Context(), businessID)
	if err != nil {
		slog.Error("Failed to get balance", "error", err, "business_id", businessID)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"balance":  balance,
		"currency": "NGN",
	})
}

// HandleGetTransactions handles GET /v1/wallets/transactions
// Gets transaction history for a business wallet
func (h *WalletHandler) HandleGetTransactions(w http.ResponseWriter, r *http.Request) {
	businessID := r.Context().Value("business_id").(uint)

	// Parse pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	transactions, total, err := h.walletService.GetTransactions(r.Context(), businessID, page, limit)
	if err != nil {
		slog.Error("Failed to get transactions", "error", err, "business_id", businessID)
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
	// Read raw body for signature verification (if needed in future)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Parse webhook payload (structure depends on KoraPay webhook format)
	// For now, we'll create a flexible structure
	var webhookPayload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &webhookPayload); err != nil {
		slog.Error("Failed to parse webhook payload", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	slog.Info("Received deposit webhook", "payload", webhookPayload)

	// Extract account reference from webhook payload
	// This depends on KoraPay's webhook structure - adjust based on actual format
	accountRef, ok := webhookPayload["account_reference"].(string)
	if !ok {
		slog.Error("Missing account_reference in webhook payload")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Find business by account reference to get businessID
	// Note: We'll need to add this method to wallet service or repository
	// For now, we'll handle it in the service layer
	
	// Parse notification from webhook payload
	notification := h.parseDepositNotification(webhookPayload, accountRef)
	if notification == nil {
		slog.Error("Failed to parse deposit notification from webhook")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Record deposit using account reference (service will look up businessID)
	if err := h.walletService.RecordDepositByAccountReference(r.Context(), accountRef, notification); err != nil {
		slog.Error("Failed to record deposit from webhook", "error", err, "account_reference", accountRef)
		// Still return 200 OK to acknowledge receipt (idempotent webhook handling)
		// Provider will retry if we return error status
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"received"}`))
		return
	}

	// Return 200 OK to acknowledge receipt
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// parseDepositNotification parses webhook payload into DepositNotification domain model
// Adjust this based on actual KoraPay webhook structure
func (h *WalletHandler) parseDepositNotification(payload map[string]interface{}, accountRef string) *domain.DepositNotification {
	// Extract fields from payload (structure depends on KoraPay webhook format)
	// This is a placeholder - adjust based on actual webhook structure
	reference, _ := payload["reference"].(string)
	if reference == "" {
		return nil
	}

	amountFloat, ok := payload["amount"].(float64)
	if !ok {
		// Try as string
		if amountStr, ok := payload["amount"].(string); ok {
			if parsed, err := strconv.ParseFloat(amountStr, 64); err == nil {
				amountFloat = parsed
			}
		}
	}

	amount := int64(amountFloat * 100) // Convert to kobo

	status, _ := payload["status"].(string)
	if status == "" {
		status = "success" // Default to success if not specified
	}

	currency, _ := payload["currency"].(string)
	if currency == "" {
		currency = "NGN"
	}

	description, _ := payload["description"].(string)

	// Parse processed_at timestamp
	processedAt := time.Now()
	if timestampStr, ok := payload["created_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			processedAt = parsed
		}
	}

	// Extract payer bank account if available
	var payerBankAccount *domain.PayerBankAccount
	if payerData, ok := payload["payer_bank_account"].(map[string]interface{}); ok {
		payerBankAccount = &domain.PayerBankAccount{
			AccountNumber: parseString(payerData["account_number"]),
			AccountName:   parseString(payerData["account_name"]),
			BankName:      parseString(payerData["bank_name"]),
		}
	}

	return &domain.DepositNotification{
		Provider:          domain.ProviderKorapay,
		Reference:         reference,
		AccountReference:  accountRef,
		Amount:            amount,
		Currency:          currency,
		Status:            status,
		Description:       description,
		ProcessedAt:       processedAt,
		PayerBankAccount:  payerBankAccount,
	}
}

// HandleSandboxCredit handles POST /v1/wallets/sandbox/credit
// Credits a virtual account in sandbox environment (testing only)
// This endpoint should only work in sandbox/test mode
func (h *WalletHandler) HandleSandboxCredit(w http.ResponseWriter, r *http.Request) {
	// Check if in sandbox mode (only allow in test/sandbox environment)
	if !h.isSandboxMode() {
		slog.Warn("Sandbox credit attempted in non-sandbox environment")
		response.RespondWithError(w, domain.ErrForbidden)
		return
	}

	businessID := r.Context().Value("business_id").(uint)

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

	// Set default currency
	if req.Currency == "" {
		req.Currency = "NGN"
	}

	// Get wallet to verify account belongs to business
	wallet, err := h.walletService.GetWallet(r.Context(), businessID)
	if err != nil {
		slog.Error("Wallet not found for sandbox credit", "error", err, "business_id", businessID)
		response.RespondWithError(w, err)
		return
	}

	// Verify account number matches
	if wallet.VirtualAccountNumber != req.AccountNumber {
		slog.Error("Account number mismatch for sandbox credit",
			"provided", req.AccountNumber,
			"wallet", wallet.VirtualAccountNumber,
			"business_id", businessID)
		response.RespondWithError(w, domain.ErrForbidden)
		return
	}

	// Call KoraPay sandbox credit API
	sandboxReq := korapay.VirtualAccountSandboxCreditRequest{
		AccountNumber: req.AccountNumber,
		Amount:        req.Amount, // Amount in main currency unit (e.g., NGN 100)
		Currency:      req.Currency,
	}

	koraResponse, err := h.koraClient.SandboxCreditVirtualAccount(sandboxReq)
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

	// Convert amount to kobo for deposit notification
	amountInKobo := int64(req.Amount * 100)

	// Manually create deposit notification (simulating webhook)
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

	// Record deposit in wallet (this will update balance)
	if err := h.walletService.RecordDeposit(r.Context(), businessID, notification); err != nil {
		slog.Error("Failed to record sandbox deposit", "error", err, "business_id", businessID)
		// Don't fail the response - credit was successful, just recording failed
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
func (h *WalletHandler) isSandboxMode() bool {
	// Check if KoraPay base URL contains sandbox/test indicators
	// This is a simple check - in production, use environment variables or config flags
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
// Creates an account holder for KYC onboarding
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

	// Convert request to domain model
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

	// Call service (service handles provider abstraction)
	result, err := h.accountHolderService.CreateAccountHolder(r.Context(), domainReq)
	if err != nil {
		slog.Error("Failed to create account holder", "error", err)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, result)
}

// HandleGetAccountHolderDetails handles GET /v1/wallets/account-holders/{reference}/details
// Retrieves account holder details by reference
func (h *WalletHandler) HandleGetAccountHolderDetails(w http.ResponseWriter, r *http.Request) {
	reference := chi.URLParam(r, "reference")
	if reference == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Call service (service handles provider abstraction)
	details, err := h.accountHolderService.GetAccountHolderDetails(r.Context(), reference)
	if err != nil {
		slog.Error("Failed to get account holder details", "error", err, "reference", reference)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, details)
}

// HandleUpdateAccountHolderKYC handles PATCH /v1/wallets/account-holders/{reference}/update-kyc
// Updates account holder KYC information
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

	// Convert request to domain model
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

	// Call service (service handles provider abstraction)
	result, err := h.accountHolderService.UpdateAccountHolderKYC(r.Context(), reference, domainReq)
	if err != nil {
		slog.Error("Failed to update account holder KYC", "error", err, "reference", reference)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, result)
}

// HandleGenerateFileUploadURL handles POST /v1/wallets/files/generate-upload-url
// Generates a pre-signed S3 URL for file uploads (KYC documents)
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

	// Convert request to domain model
	domainReq := &domain.GenerateFileUploadURLRequest{
		Reference:   req.Reference,
		Purpose:     req.Purpose,
		ContentType: req.ContentType,
	}

	// Call service (service handles provider abstraction)
	result, err := h.accountHolderService.GenerateFileUploadURL(r.Context(), domainReq)
	if err != nil {
		slog.Error("Failed to generate file upload URL", "error", err)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, result)
}
