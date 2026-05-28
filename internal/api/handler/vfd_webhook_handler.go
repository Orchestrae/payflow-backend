package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
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

type VFDWebhookHandler struct {
	webhookService service.VFDWebhookService
	webhookSecret  string
	validate       *validator.Validate
}

func NewVFDWebhookHandler(svc service.VFDWebhookService, webhookSecret string) *VFDWebhookHandler {
	return &VFDWebhookHandler{
		webhookService: svc,
		webhookSecret:  webhookSecret,
		validate:       validator.New(),
	}
}

// verifySignature verifies VFD webhook HMAC-SHA256 signature.
func (h *VFDWebhookHandler) verifySignature(body []byte, signature string) bool {
	if h.webhookSecret == "" {
		return true // Skip verification if no secret configured
	}
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

// HandleInwardCreditWebhook handles POST /vfd/webhooks/inward-credit
// This endpoint receives webhook notifications for settled inward credit transactions
func (h *VFDWebhookHandler) HandleInwardCreditWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Verify HMAC signature if secret is configured
	if h.webhookSecret != "" {
		signature := r.Header.Get("X-VFD-Signature")
		if signature == "" || !h.verifySignature(bodyBytes, signature) {
			slog.Error("VFD webhook signature verification failed")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req request.VFDInwardCreditWebhookRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		slog.Error("Failed to decode inward credit webhook request", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		slog.Error("Inward credit webhook validation failed", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Convert request to domain model
	notification := &domain.VFDWebhookNotification{
		Reference:               req.Reference,
		Amount:                  req.Amount,
		AccountNumber:           req.AccountNumber,
		OriginatorAccountNumber: req.OriginatorAccountNumber,
		OriginatorAccountName:   req.OriginatorAccountName,
		OriginatorBank:          req.OriginatorBank,
		OriginatorNarration:     req.OriginatorNarration,
		Timestamp:               req.Timestamp,
		TransactionChannel:      req.TransactionChannel,
		SessionID:               req.SessionID,
		InitialCreditRequest:    false, // This is a settled transaction
	}

	// Process the webhook
	if err := h.webhookService.ProcessInwardCreditWebhook(r.Context(), notification); err != nil {
		slog.Error("Failed to process inward credit webhook", "error", err, "reference", req.Reference)
		response.RespondWithError(w, err)
		return
	}

	// Return 200 OK as required by VFD
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// HandleInitialInwardCreditWebhook handles POST /vfd/webhooks/initial-inward-credit
// This endpoint receives webhook notifications for initial inward credit transactions (before settlement)
func (h *VFDWebhookHandler) HandleInitialInwardCreditWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if h.webhookSecret != "" {
		signature := r.Header.Get("X-VFD-Signature")
		if signature == "" || !h.verifySignature(bodyBytes, signature) {
			slog.Error("VFD webhook signature verification failed")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req request.VFDInitialInwardCreditWebhookRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		slog.Error("Failed to decode initial inward credit webhook request", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		slog.Error("Initial inward credit webhook validation failed", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Convert request to domain model
	notification := &domain.VFDWebhookNotification{
		Reference:               req.Reference,
		Amount:                  req.Amount,
		AccountNumber:           req.AccountNumber,
		OriginatorAccountNumber: req.OriginatorAccountNumber,
		OriginatorAccountName:   req.OriginatorAccountName,
		OriginatorBank:          req.OriginatorBank,
		OriginatorNarration:     req.OriginatorNarration,
		Timestamp:               req.Timestamp,
		TransactionChannel:      "", // Not provided in initial credit
		SessionID:               req.SessionID,
		InitialCreditRequest:    req.InitialCreditRequest,
	}

	// Process the webhook
	if err := h.webhookService.ProcessInitialInwardCreditWebhook(r.Context(), notification); err != nil {
		slog.Error("Failed to process initial inward credit webhook", "error", err, "reference", req.Reference)
		response.RespondWithError(w, err)
		return
	}

	// Return 200 OK as required by VFD
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// RetriggerWebhook handles POST /vfd/webhooks/retrigger
// This endpoint allows retriggering webhook notifications via VFD API
func (h *VFDWebhookHandler) RetriggerWebhook(w http.ResponseWriter, r *http.Request) {
	var req request.VFDRetriggerWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode retrigger webhook request", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		slog.Error("Retrigger webhook validation failed", "error", err)
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Convert request to domain model
	retriggerReq := &domain.VFDRetriggerRequest{
		TransactionID:  req.TransactionID,
		SessionID:      req.SessionID,
		PushIdentifier: req.PushIdentifier,
	}

	// Call VFD API to retrigger webhook
	retriggerResponse, err := h.webhookService.RetriggerWebhookNotification(r.Context(), retriggerReq)
	if err != nil {
		slog.Error("Failed to retrigger webhook", "error", err, "transaction_id", req.TransactionID, "session_id", req.SessionID)
		response.RespondWithError(w, err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, retriggerResponse)
}

// ListWebhookNotifications handles GET /vfd/webhooks
// This endpoint lists webhook notifications for a business
func (h *VFDWebhookHandler) ListWebhookNotifications(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, domain.ErrUnauthorized)
		return
	}
	businessID := uint64(claims.BusinessID)

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get webhook notifications
	notifications, total, err := h.webhookService.ListWebhookNotifications(r.Context(), uint(businessID), page, limit)
	if err != nil {
		slog.Error("Failed to list webhook notifications", "error", err, "business_id", businessID)
		response.RespondWithError(w, err)
		return
	}

	// Convert to response DTOs
	responseDTOs := make([]response.VFDWebhookNotificationResponse, len(notifications))
	for i, notification := range notifications {
		responseDTOs[i] = response.VFDWebhookNotificationResponse{
			ID:                      notification.ID,
			Reference:               notification.Reference,
			Amount:                  notification.Amount,
			AccountNumber:           notification.AccountNumber,
			OriginatorAccountNumber: notification.OriginatorAccountNumber,
			OriginatorAccountName:   notification.OriginatorAccountName,
			OriginatorBank:          notification.OriginatorBank,
			OriginatorNarration:     notification.OriginatorNarration,
			Timestamp:               notification.Timestamp,
			TransactionChannel:      notification.TransactionChannel,
			SessionID:               notification.SessionID,
			InitialCreditRequest:    notification.InitialCreditRequest,
			Status:                  notification.Status,
			ProcessedAt:             notification.ProcessedAt,
			ProcessingError:         notification.ProcessingError,
			CreatedAt:               notification.CreatedAt,
		}
	}

	listResponse := response.VFDWebhookListResponse{
		Notifications: responseDTOs,
		Total:         total,
		Page:          page,
		Limit:         limit,
	}

	response.RespondWithJSON(w, http.StatusOK, listResponse)
}

// GetWebhookNotificationByID handles GET /vfd/webhooks/{id}
// This endpoint gets a specific webhook notification by ID
func (h *VFDWebhookHandler) GetWebhookNotificationByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	notification, err := h.webhookService.GetWebhookNotificationByID(r.Context(), uint(id))
	if err != nil {
		slog.Error("Failed to get webhook notification", "error", err, "id", id)
		response.RespondWithError(w, err)
		return
	}

	responseDTO := response.VFDWebhookNotificationResponse{
		ID:                      notification.ID,
		Reference:               notification.Reference,
		Amount:                  notification.Amount,
		AccountNumber:           notification.AccountNumber,
		OriginatorAccountNumber: notification.OriginatorAccountNumber,
		OriginatorAccountName:   notification.OriginatorAccountName,
		OriginatorBank:          notification.OriginatorBank,
		OriginatorNarration:     notification.OriginatorNarration,
		Timestamp:               notification.Timestamp,
		TransactionChannel:      notification.TransactionChannel,
		SessionID:               notification.SessionID,
		InitialCreditRequest:    notification.InitialCreditRequest,
		Status:                  notification.Status,
		ProcessedAt:             notification.ProcessedAt,
		ProcessingError:         notification.ProcessingError,
		CreatedAt:               notification.CreatedAt,
	}

	response.RespondWithJSON(w, http.StatusOK, responseDTO)
}

// GetWebhookNotificationsByAccountNumber handles GET /vfd/webhooks/account/{accountNumber}
// This endpoint gets webhook notifications for a specific account number
func (h *VFDWebhookHandler) GetWebhookNotificationsByAccountNumber(w http.ResponseWriter, r *http.Request) {
	accountNumber := chi.URLParam(r, "accountNumber")
	if accountNumber == "" {
		response.RespondWithError(w, domain.ErrValidationFailed)
		return
	}

	// Get query parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get webhook notifications
	notifications, total, err := h.webhookService.GetWebhookNotificationsByAccountNumber(r.Context(), accountNumber, page, limit)
	if err != nil {
		slog.Error("Failed to get webhook notifications by account number", "error", err, "account_number", accountNumber)
		response.RespondWithError(w, err)
		return
	}

	// Convert to response DTOs
	responseDTOs := make([]response.VFDWebhookNotificationResponse, len(notifications))
	for i, notification := range notifications {
		responseDTOs[i] = response.VFDWebhookNotificationResponse{
			ID:                      notification.ID,
			Reference:               notification.Reference,
			Amount:                  notification.Amount,
			AccountNumber:           notification.AccountNumber,
			OriginatorAccountNumber: notification.OriginatorAccountNumber,
			OriginatorAccountName:   notification.OriginatorAccountName,
			OriginatorBank:          notification.OriginatorBank,
			OriginatorNarration:     notification.OriginatorNarration,
			Timestamp:               notification.Timestamp,
			TransactionChannel:      notification.TransactionChannel,
			SessionID:               notification.SessionID,
			InitialCreditRequest:    notification.InitialCreditRequest,
			Status:                  notification.Status,
			ProcessedAt:             notification.ProcessedAt,
			ProcessingError:         notification.ProcessingError,
			CreatedAt:               notification.CreatedAt,
		}
	}

	listResponse := response.VFDWebhookListResponse{
		Notifications: responseDTOs,
		Total:         total,
		Page:          page,
		Limit:         limit,
	}

	response.RespondWithJSON(w, http.StatusOK, listResponse)
}
