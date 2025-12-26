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

// SingleDisbursementResponse represents the response from a single disbursement
type SingleDisbursementResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Reference string `json:"reference,omitempty"`
		Amount    string `json:"amount,omitempty"`
		Currency  string `json:"currency,omitempty"`
		Status    string `json:"status,omitempty"`
	} `json:"data,omitempty"`
}

// BulkPayoutItem represents a single payout item in a bulk disbursement
type BulkPayoutItem struct {
	Reference   string                  `json:"reference"`
	Amount      float64                 `json:"amount"` // Amount as number
	Type        string                  `json:"type"`   // "bank_account" or "mobile_money"
	Narration   string                  `json:"narration"`
	BankAccount *BankAccountDestination `json:"bank_account,omitempty"`
	MobileMoney *MobileMoneyDestination `json:"mobile_money,omitempty"`
	Customer    Customer                `json:"customer"`
}

// BulkPayoutRequest represents a bulk disbursement request
type BulkPayoutRequest struct {
	BatchReference    string           `json:"batch_reference"`
	Description       string           `json:"description,omitempty"`
	MerchantBearsCost bool             `json:"merchant_bears_cost,omitempty"`
	Currency          string           `json:"currency"`
	Payouts           []BulkPayoutItem `json:"payouts"`
}

// BulkPayoutResponse represents the response from a bulk disbursement
type BulkPayoutResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Reference string `json:"reference,omitempty"`
	} `json:"data,omitempty"`
}

// Legacy types for backward compatibility (deprecated - use new types above)
type BulkPayoutDestination struct {
	BankAccount string  `json:"bank_account"`
	BankCode    string  `json:"bank_code"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
}
