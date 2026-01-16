package request

// SingleTransferRequest represents a simple transfer request from a business user.
// Only essential business information is required - everything else is handled internally.
type SingleTransferRequest struct {
	// Required: Who to pay and how much
	Amount        string `json:"amount" validate:"required"`
	BankCode      string `json:"bank_code" validate:"required"`
	AccountNumber string `json:"account_number" validate:"required"`
	AccountName   string `json:"account_name" validate:"required"`

	// Optional: Why (defaults to "Transfer" if not provided)
	Narration string `json:"narration,omitempty"`

	// Optional: Custom reference (auto-generated if not provided)
	// Business users typically don't need to provide this
	Reference string `json:"reference,omitempty"`
}

// BatchTransferRequest represents a batch of transfer requests.
// Each transfer in the batch follows the same simple format.
type BatchTransferRequest struct {
	Transfers []SingleTransferRequest `json:"transfers" validate:"required,min=1,max=100,dive"`
}
