// internal/platform/korapay/types.go
package korapay

// This file contains structs that map directly to the KoraPay API.
// Based on actual API: https://api.korapay.com/merchant/api/v1/transactions/disburse

// Customer represents customer information in disbursement requests
type Customer struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// BankAccountDestination represents bank account destination details
type BankAccountDestination struct {
	Bank    string `json:"bank"`    // Bank code (e.g., "033", "044")
	Account string `json:"account"` // Account number
}

// MobileMoneyDestination represents mobile money destination details
type MobileMoneyDestination struct {
	Operator     string `json:"operator"`      // e.g., "safaricom-ke"
	MobileNumber string `json:"mobile_number"` // e.g., "256700000000"
}

// DisbursementDestination represents the destination for a disbursement
type DisbursementDestination struct {
	Type        string                  `json:"type"`                   // "bank_account" or "mobile_money"
	Amount      string                  `json:"amount"`                 // Amount as string
	Currency    string                  `json:"currency"`               // e.g., "NGN", "KES"
	Narration   string                  `json:"narration"`              // Transfer description
	BankAccount *BankAccountDestination `json:"bank_account,omitempty"` // For bank_account type
	MobileMoney *MobileMoneyDestination `json:"mobile_money,omitempty"` // For mobile_money type
	Customer    Customer                `json:"customer"`               // Customer information
}

// SingleDisbursementRequest represents a single disbursement request
type SingleDisbursementRequest struct {
	Reference   string                  `json:"reference"`
	Destination DisbursementDestination `json:"destination"`
}

// SingleDisbursementData represents the data in a single disbursement response
type SingleDisbursementData struct {
	Reference string `json:"reference,omitempty"`
	Amount    string `json:"amount,omitempty"`
	Fee       string `json:"fee,omitempty"`
	Currency  string `json:"currency,omitempty"`
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
}

// SingleDisbursementResponse represents the response from a single disbursement
type SingleDisbursementResponse struct {
	Status  bool                    `json:"status"` // Korapay returns boolean, not string
	Message string                  `json:"message"`
	Data    *SingleDisbursementData `json:"data,omitempty"`
}

// ============================================================================
// Bulk Payout Types (matches Korapay API exactly)
// ============================================================================

// BulkBankAccountDestination represents bank account destination for BULK payouts (different field names!)
type BulkBankAccountDestination struct {
	BankCode      string `json:"bank_code"`      // Bank code (e.g., "033", "044")
	AccountNumber string `json:"account_number"` // Account number
}

// BulkPayoutItem represents a single payout item in a bulk disbursement
type BulkPayoutItem struct {
	Reference   string                      `json:"reference"`
	Amount      float64                     `json:"amount"` // Amount as number (two decimal places)
	Type        string                      `json:"type"`   // "bank_account" or "mobile_money"
	Narration   string                      `json:"narration"`
	BankAccount *BulkBankAccountDestination `json:"bank_account,omitempty"` // Uses different field names than single!
	MobileMoney *MobileMoneyDestination     `json:"mobile_money,omitempty"`
	Customer    Customer                    `json:"customer"`
}

// BulkPayoutRequest represents a bulk disbursement request
type BulkPayoutRequest struct {
	BatchReference    string           `json:"batch_reference"`              // Required, 5-50 chars
	Description       string           `json:"description,omitempty"`        // Optional
	MerchantBearsCost bool             `json:"merchant_bears_cost,omitempty"` // Optional, defaults to false
	Currency          string           `json:"currency"`                      // Required: NGN, KES, GHS, ZAR
	Payouts           []BulkPayoutItem `json:"payouts"`                       // Required: 2-50 items
}

// BulkPayoutResponseData represents the data in a bulk payout response
type BulkPayoutResponseData struct {
	Status               string  `json:"status"`                 // pending, processing, complete, failed
	TotalChargeableAmount float64 `json:"total_chargeable_amount,omitempty"`
	MerchantBearsCost    bool    `json:"merchant_bears_cost,omitempty"`
	Reference            string  `json:"reference,omitempty"`
	Currency             string  `json:"currency,omitempty"`
	Description          string  `json:"description,omitempty"`
	CreatedAt            string  `json:"createdAt,omitempty"`
	// For fetch bulk payout
	Amount                 string `json:"amount,omitempty"`
	FailedTransactions     int    `json:"failedTransactions,omitempty"`
	SuccessfulTransactions int    `json:"successfulTransactions,omitempty"`
	PendingTransactions    int    `json:"pendingTransactions,omitempty"`
	ProcessingTransactions int    `json:"processingTransactions,omitempty"`
}

// BulkPayoutResponse represents the response from a bulk disbursement
type BulkPayoutResponse struct {
	Status  bool                    `json:"status"` // Boolean like single disbursement
	Message string                  `json:"message"`
	Data    *BulkPayoutResponseData `json:"data,omitempty"`
}

// BulkPayoutItemStatus represents a single payout status in a batch
type BulkPayoutItemStatus struct {
	Reference      string   `json:"reference"`
	Amount         float64  `json:"amount"`
	Currency       string   `json:"currency"`
	Narration      string   `json:"narration"`
	Status         string   `json:"status"` // success, failed, pending, processing
	BatchReference string   `json:"batch_reference"`
	Type           string   `json:"type"`
	Customer       Customer `json:"customer"`
	BankAccount    *BankAccountDestination `json:"bank_account,omitempty"`
	Fee            float64  `json:"fee,omitempty"`
	Message        string   `json:"message,omitempty"`
	TraceID        string   `json:"trace_id,omitempty"`
}

// BulkPayoutPayoutsResponse represents the response from fetching payouts in a batch
type BulkPayoutPayoutsResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Data   []BulkPayoutItemStatus `json:"data"`
		Paging struct {
			TotalItems int `json:"total_items"`
			PageSize   int `json:"page_size"`
			Current    int `json:"current"`
			Count      int `json:"count"`
		} `json:"paging"`
	} `json:"data,omitempty"`
}
