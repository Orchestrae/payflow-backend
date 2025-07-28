package domain

import (
	"time"
)

// VFDWebhookNotification represents a webhook notification from VFD
type VFDWebhookNotification struct {
	Model
	BusinessID              uint   `gorm:"index"`
	Reference               string `gorm:"size:255;index"`
	Amount                  string `gorm:"size:50"`
	AccountNumber           string `gorm:"size:20;index"`
	OriginatorAccountNumber string `gorm:"size:20"`
	OriginatorAccountName   string `gorm:"size:255"`
	OriginatorBank          string `gorm:"size:10"`
	OriginatorNarration     string `gorm:"size:500"`
	Timestamp               time.Time
	TransactionChannel      string `gorm:"size:10"`
	SessionID               string `gorm:"size:50;index"`
	InitialCreditRequest    bool   `gorm:"default:false"`
	Status                  string `gorm:"size:20;default:'pending'"` // pending, processed, failed
	ProcessedAt             *time.Time
	ProcessingError         *string `gorm:"size:1000"`

	// Relationships
	Business *Business `gorm:"-"`
}

// VFDRetriggerRequest represents a request to retrigger a webhook notification
type VFDRetriggerRequest struct {
	TransactionID  string `json:"transactionId" validate:"required_without=SessionID"`
	SessionID      string `json:"sessionId" validate:"required_without=TransactionID"`
	PushIdentifier string `json:"pushIdentifier" validate:"required,oneof=transactionId sessionId"`
}

// VFDRetriggerResponse represents the response from VFD retrigger endpoint
type VFDRetriggerResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// WebhookNotificationType represents the type of webhook notification
type WebhookNotificationType string

const (
	WebhookTypeInwardCredit        WebhookNotificationType = "inward_credit"
	WebhookTypeInitialInwardCredit WebhookNotificationType = "initial_inward_credit"
)

// WebhookStatus represents the processing status of a webhook
type WebhookStatus string

const (
	WebhookStatusPending   WebhookStatus = "pending"
	WebhookStatusProcessed WebhookStatus = "processed"
	WebhookStatusFailed    WebhookStatus = "failed"
)
