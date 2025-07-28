package response

import "time"

// VFDWebhookNotificationResponse represents the response for webhook notifications
type VFDWebhookNotificationResponse struct {
	ID                      uint       `json:"id"`
	Reference               string     `json:"reference"`
	Amount                  string     `json:"amount"`
	AccountNumber           string     `json:"account_number"`
	OriginatorAccountNumber string     `json:"originator_account_number"`
	OriginatorAccountName   string     `json:"originator_account_name"`
	OriginatorBank          string     `json:"originator_bank"`
	OriginatorNarration     string     `json:"originator_narration"`
	Timestamp               time.Time  `json:"timestamp"`
	TransactionChannel      string     `json:"transaction_channel"`
	SessionID               string     `json:"session_id"`
	InitialCreditRequest    bool       `json:"initial_credit_request"`
	Status                  string     `json:"status"`
	ProcessedAt             *time.Time `json:"processed_at,omitempty"`
	ProcessingError         *string    `json:"processing_error,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
}

// VFDRetriggerResponse represents the response from VFD retrigger endpoint
type VFDRetriggerResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// VFDWebhookListResponse represents the response for listing webhook notifications
type VFDWebhookListResponse struct {
	Notifications []VFDWebhookNotificationResponse `json:"notifications"`
	Total         int                              `json:"total"`
	Page          int                              `json:"page"`
	Limit         int                              `json:"limit"`
}
