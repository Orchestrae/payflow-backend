package domain

import "time"

// ProviderName represents the name of a transfer provider
type ProviderName string

const (
	ProviderKorapay ProviderName = "korapay"
	ProviderVFD     ProviderName = "vfd"
)

// SingleTransferRequest represents a provider-agnostic request to transfer funds.
// This is the unified internal format that all providers can work with.
type SingleTransferRequest struct {
	// Core transfer details (from user input)
	Reference     string `json:"reference"`
	Amount        string `json:"amount"`
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	Narration     string `json:"narration"`
	Currency      string `json:"currency,omitempty"` // Defaults to NGN

	// Internal fields (populated by service layer, not from user input)
	BusinessID    uint   `json:"-"` // Set by service from auth context
	BusinessEmail string `json:"-"` // Set by service for provider-specific needs
}

// TransferResult represents the unified result from any transfer provider.
type TransferResult struct {
	Success       bool         `json:"success"`
	Reference     string       `json:"reference"`
	TransactionID string       `json:"transaction_id,omitempty"`
	Status        string       `json:"status"` // processing, success, failed
	Message       string       `json:"message"`
	Provider      ProviderName `json:"provider"`
	Fee           string       `json:"fee,omitempty"`
	Currency      string       `json:"currency,omitempty"`
}

// Transfer represents a transfer record in the database.
// This is provider-agnostic and stores the essential information.
type Transfer struct {
	Model

	// Business relationship
	BusinessID uint `gorm:"index"`

	// Core transfer details
	Reference string `gorm:"size:100;uniqueIndex"`
	Amount    string `gorm:"size:20"`
	Currency  string `gorm:"size:10;default:'NGN'"`
	Narration string `gorm:"size:500"`

	// Recipient details
	RecipientBankCode      string `gorm:"size:10"`
	RecipientAccountNumber string `gorm:"size:20;index"`
	RecipientAccountName   string `gorm:"size:255"`

	// Provider and status
	Provider        string `gorm:"size:20"`                   // korapay, vfd, etc.
	Status          string `gorm:"size:20;default:'pending'"` // pending, processing, success, failed
	TransactionID   string `gorm:"size:100"`                  // Provider's transaction ID
	ProviderStatus  string `gorm:"size:50"`                   // Provider-specific status
	ProviderMessage string `gorm:"size:500"`                  // Provider-specific message
	Fee             string `gorm:"size:20"`

	// Processing tracking
	ProcessedAt     *time.Time
	ProcessingError *string `gorm:"size:1000"`

	// Relationships
	Business *Business `gorm:"-"`
}

// TableName specifies the table name for GORM
func (Transfer) TableName() string {
	return "transfers"
}

// SingleTransferResponse is the response returned to API clients.
type SingleTransferResponse struct {
	Success        bool          `json:"success"`
	TransferID     uint          `json:"transfer_id,omitempty"`
	Reference      string        `json:"reference"`
	TransactionID  string        `json:"transaction_id,omitempty"`
	Status         string        `json:"status"`
	Message        string        `json:"message,omitempty"`
	Provider       ProviderName  `json:"provider,omitempty"`
	Fee            string        `json:"fee,omitempty"`
	ProcessingTime time.Duration `json:"processing_time"`
	Error          string        `json:"error,omitempty"`
}

// ============================================================================
// Bulk Transfer Types
// ============================================================================

// BulkTransferRequest represents a batch of transfer requests.
// Used both for API requests and internal processing.
type BulkTransferRequest struct {
	// BatchReference is a unique identifier for the entire batch (auto-generated if empty)
	BatchReference string `json:"batch_reference,omitempty"`

	// Description is an optional description for the batch
	Description string `json:"description,omitempty"`

	// Currency for all transfers in the batch (defaults to NGN)
	Currency string `json:"currency,omitempty"`

	// MerchantBearsCost determines who pays the fees (provider-specific)
	MerchantBearsCost bool `json:"merchant_bears_cost,omitempty"`

	// Transfers is the list of individual transfers in the batch
	Transfers []SingleTransferRequest `json:"transfers" validate:"required,min=1,max=100"`

	// Internal fields (set by service layer)
	BusinessID    uint   `json:"-"`
	BusinessEmail string `json:"-"`
}

// BulkTransferResult represents the result from a bulk transfer operation.
// This is the provider-level result (before service processing).
type BulkTransferResult struct {
	Success        bool         `json:"success"`
	BatchReference string       `json:"batch_reference"`
	Provider       ProviderName `json:"provider"`
	Status         string       `json:"status"` // pending, processing, complete, failed
	Message        string       `json:"message,omitempty"`
	Currency       string       `json:"currency,omitempty"`

	// Aggregate counts (if available from provider)
	TotalAmount          string `json:"total_amount,omitempty"`
	TotalChargeableAmount string `json:"total_chargeable_amount,omitempty"`

	// Individual transfer results (populated for providers that return them)
	TransferResults []TransferResult `json:"transfer_results,omitempty"`
}

// BulkTransferResponse represents the API response from a batch transfer operation.
// This is what the API returns to clients.
type BulkTransferResponse struct {
	Success             bool                     `json:"success"`
	BatchReference      string                   `json:"batch_reference"`
	TotalTransfers      int                      `json:"total_transfers"`
	SuccessfulTransfers int                      `json:"successful_transfers"`
	FailedTransfers     int                      `json:"failed_transfers"`
	PendingTransfers    int                      `json:"pending_transfers"`
	Provider            ProviderName             `json:"provider,omitempty"`
	Transfers           []SingleTransferResponse `json:"transfers"`
	ProcessingTime      time.Duration            `json:"processing_time"`
}

