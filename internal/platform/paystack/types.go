package paystack

// CreateRecipientRequest represents a Paystack transfer recipient creation request.
type CreateRecipientRequest struct {
	Type          string `json:"type"`           // "nuban" for Nigerian bank accounts
	Name          string `json:"name"`           // recipient name
	AccountNumber string `json:"account_number"` // bank account number
	BankCode      string `json:"bank_code"`      // bank code (e.g., "058" for GTBank)
	Currency      string `json:"currency"`       // "NGN"
}

// CreateRecipientResponse represents the Paystack transfer recipient creation response.
type CreateRecipientResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    *RecipientData `json:"data"`
}

// RecipientData contains the created recipient details.
type RecipientData struct {
	RecipientCode string `json:"recipient_code"` // e.g., "RCP_gx2wn530m0i3w3m"
	Name          string `json:"name"`
	Type          string `json:"type"`
}

// TransferRequest represents a Paystack single transfer request.
type TransferRequest struct {
	Source    string `json:"source"`    // "balance"
	Amount   int64  `json:"amount"`    // amount in kobo
	Recipient string `json:"recipient"` // recipient_code from CreateRecipient
	Reference string `json:"reference"` // unique reference
	Reason   string `json:"reason"`    // narration
	Currency string `json:"currency"`  // "NGN"
}

// TransferResponse represents the Paystack single transfer response.
type TransferResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    *TransferData `json:"data"`
}

// TransferData contains the transfer details.
type TransferData struct {
	TransferCode string `json:"transfer_code"`
	Reference    string `json:"reference"`
	Status       string `json:"status"` // "success", "pending", "failed"
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
}

// BulkTransferRequest represents a Paystack bulk transfer request.
type BulkTransferRequest struct {
	Source    string              `json:"source"`    // "balance"
	Currency string              `json:"currency"`  // "NGN"
	Transfers []BulkTransferItem `json:"transfers"`
}

// BulkTransferItem represents a single item in a bulk transfer.
type BulkTransferItem struct {
	Amount    int64  `json:"amount"`    // kobo
	Recipient string `json:"recipient"` // recipient_code
	Reference string `json:"reference"`
	Reason    string `json:"reason"`
}

// BulkTransferResponse represents the Paystack bulk transfer response.
type BulkTransferResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

// ResolveAccountResponse represents the Paystack bank resolve response.
type ResolveAccountResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    *ResolveAccountData `json:"data"`
}

// ResolveAccountData contains the resolved account details.
type ResolveAccountData struct {
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	BankID        int    `json:"bank_id"`
}

// ResolveBVNResponse represents the Paystack BVN resolve response.
type ResolveBVNResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    *BVNData `json:"data"`
}

// BVNData contains the resolved BVN details.
type BVNData struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DOB         string `json:"dob"`
	Phone       string `json:"formatted_dob"`
	BVN         string `json:"bvn"`
}

// WebhookPayload represents a Paystack webhook event payload.
type WebhookPayload struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data"`
}
