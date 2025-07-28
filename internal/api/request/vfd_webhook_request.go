package request

import "time"

// VFDInwardCreditWebhookRequest represents the webhook payload for inward credit notifications
type VFDInwardCreditWebhookRequest struct {
	Reference               string    `json:"reference" validate:"required"`
	Amount                  string    `json:"amount" validate:"required"`
	AccountNumber           string    `json:"account_number" validate:"required"`
	OriginatorAccountNumber string    `json:"originator_account_number" validate:"required"`
	OriginatorAccountName   string    `json:"originator_account_name" validate:"required"`
	OriginatorBank          string    `json:"originator_bank" validate:"required"`
	OriginatorNarration     string    `json:"originator_narration"`
	Timestamp               time.Time `json:"timestamp" validate:"required"`
	TransactionChannel      string    `json:"transaction_channel" validate:"required"`
	SessionID               string    `json:"session_id" validate:"required"`
}

// VFDInitialInwardCreditWebhookRequest represents the webhook payload for initial inward credit notifications
type VFDInitialInwardCreditWebhookRequest struct {
	Reference               string    `json:"reference" validate:"required"`
	Amount                  string    `json:"amount" validate:"required"`
	AccountNumber           string    `json:"account_number" validate:"required"`
	OriginatorAccountNumber string    `json:"originator_account_number" validate:"required"`
	OriginatorAccountName   string    `json:"originator_account_name" validate:"required"`
	OriginatorBank          string    `json:"originator_bank" validate:"required"`
	OriginatorNarration     string    `json:"originator_narration"`
	Timestamp               time.Time `json:"timestamp" validate:"required"`
	SessionID               string    `json:"session_id" validate:"required"`
	InitialCreditRequest    bool      `json:"initialCreditRequest"`
}

// VFDRetriggerWebhookRequest represents the request to retrigger a webhook notification
type VFDRetriggerWebhookRequest struct {
	TransactionID  string `json:"transactionId" validate:"required_without=SessionID"`
	SessionID      string `json:"sessionId" validate:"required_without=TransactionID"`
	PushIdentifier string `json:"pushIdentifier" validate:"required,oneof=transactionId sessionId"`
}
