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

// ============================================================================
// Virtual Bank Account Types (matches Korapay API exactly)
// ============================================================================

// VirtualAccountKYC represents KYC information for virtual account creation
type VirtualAccountKYC struct {
	BVN string `json:"bvn"`           // Required: 11 digits
	NIN string `json:"nin,omitempty"` // Optional: National Identification Number
}

// VirtualAccountCustomer represents customer information for virtual account
type VirtualAccountCustomer struct {
	Name  string `json:"name"`            // Required
	Email string `json:"email,omitempty"` // Optional
}

// VirtualAccountCreateRequest represents the request to create a virtual account
type VirtualAccountCreateRequest struct {
	AccountName     string                `json:"account_name"`     // Required: Name for the virtual account
	AccountReference string                `json:"account_reference"` // Required: Your unique reference (5-50 chars)
	Permanent       bool                  `json:"permanent"`        // Required: Must be true (temporary accounts not supported)
	BankCode        string                `json:"bank_code"`        // Required: Bank code (e.g., "000" for sandbox, "035" for Wema)
	Customer        VirtualAccountCustomer `json:"customer"`        // Required: Customer information
	KYC             VirtualAccountKYC     `json:"kyc"`             // Required: KYC information
}

// VirtualAccountData represents the data in a virtual account response
type VirtualAccountData struct {
	AccountName      string                `json:"account_name"`
	AccountNumber    string                `json:"account_number"`
	BankCode         string                `json:"bank_code"`
	BankName         string                `json:"bank_name"`
	Customer         VirtualAccountCustomer `json:"customer"`
	AccountReference string                `json:"account_reference"`
	UniqueID         string                `json:"unique_id"`         // e.g., "KPY-VA-yrJtnFgVesLeKgM"
	AccountStatus    string                `json:"account_status"`    // active, suspended, inactive
	CreatedAt        string                `json:"created_at"`        // ISO 8601 format
	Currency         string                `json:"currency"`          // NGN
}

// VirtualAccountResponse represents the response from creating/getting a virtual account
type VirtualAccountResponse struct {
	Status  bool                `json:"status"`  // Boolean
	Message string              `json:"message"`
	Data    *VirtualAccountData `json:"data,omitempty"`
}

// VirtualAccountTransactionItem represents a single transaction in a virtual account
type VirtualAccountTransactionItem struct {
	Reference        string `json:"reference"`        // e.g., "KPY-PAY-4l5O8mxmgX2kijp"
	Status           string `json:"status"`           // success, pending, failed
	Amount           string `json:"amount"`           // Amount as string (e.g., "1000.00")
	Fee              string `json:"fee,omitempty"`    // Fee as string (e.g., "15.00")
	Currency         string `json:"currency"`         // NGN
	Description      string `json:"description"`      // e.g., "Payment by Don Alpha"
	PayerBankAccount *PayerBankAccountDetails `json:"payer_bank_account,omitempty"`
}

// PayerBankAccountDetails represents payer bank account details in a transaction
type PayerBankAccountDetails struct {
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	BankName      string `json:"bank_name"`
}

// VirtualAccountTransactionsData represents the data in a virtual account transactions response
type VirtualAccountTransactionsData struct {
	TotalAmountReceived int64                       `json:"total_amount_received"` // Total as integer (kobo)
	AccountNumber       string                      `json:"account_number"`
	Currency            string                      `json:"currency"`
	Transactions        []VirtualAccountTransactionItem `json:"transactions"`
	Pagination          VirtualAccountPagination    `json:"pagination"`
}

// VirtualAccountPagination represents pagination info in transactions response
type VirtualAccountPagination struct {
	Page       int `json:"page"`
	Total      int `json:"total"`
	PageCount  int `json:"pageCount"`
	TotalPages int `json:"totalPages"`
}

// VirtualAccountTransactionsResponse represents the response from getting virtual account transactions
type VirtualAccountTransactionsResponse struct {
	Status  bool                           `json:"status"`
	Message string                         `json:"message"`
	Data    *VirtualAccountTransactionsData `json:"data,omitempty"`
}

// VirtualAccountSandboxCreditRequest represents the request to credit a virtual account in sandbox
type VirtualAccountSandboxCreditRequest struct {
	AccountNumber string `json:"account_number"` // Virtual account number
	Amount        int    `json:"amount"`         // Amount as integer (kobo)
	Currency      string `json:"currency"`       // NGN
}

// VirtualAccountSandboxCreditResponse represents the response from sandbox credit
type VirtualAccountSandboxCreditResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"` // Usually null for this endpoint
}

// ============================================================================
// Account Holder Types (KYC/Onboarding)
// ============================================================================

// FileReference represents a file reference for document uploads
type FileReference struct {
	Reference string `json:"reference"`
}

// AccountHolderIdentification represents identification document details
type AccountHolderIdentification struct {
	Type         string        `json:"type"`          // passport, national_id, driver_license, etc.
	Number       string        `json:"number"`
	DocumentFront *FileReference `json:"document_front,omitempty"`
	DocumentBack  *FileReference `json:"document_back,omitempty"`
	IssuedDate   string        `json:"issued_date,omitempty"`   // YYYY-MM-DD
	ExpiryDate   string        `json:"expiry_date,omitempty"`   // YYYY-MM-DD
	Country      string        `json:"country,omitempty"`       // NG, etc.
}

// AccountHolderProofOfAddress represents proof of address document
type AccountHolderProofOfAddress struct {
	Type     string         `json:"type"`     // bank_statement, utility_bill, etc.
	Document *FileReference `json:"document,omitempty"`
}

// AccountHolderAddress represents physical address details
type AccountHolderAddress struct {
	Country string `json:"country"` // NG
	Zip     string `json:"zip,omitempty"`
	Address string `json:"address"` // Street address
	State   string `json:"state"`
	City    string `json:"city"`
}

// AccountHolderEmployment represents employment details
type AccountHolderEmployment struct {
	Status      string `json:"status"`       // employer, employee, self_employed, unemployed, student
	Employer    string `json:"employer,omitempty"`
	Description string `json:"description,omitempty"`
}

// AccountHolderCreateRequest represents the request to create an account holder
type AccountHolderCreateRequest struct {
	FirstName           string                       `json:"first_name"`
	LastName            string                       `json:"last_name"`
	UseCase             string                       `json:"use_case"`              // Personal, Business
	Type                string                       `json:"type"`                  // individual, business
	DateOfBirth         string                       `json:"date_of_birth"`         // YYYY-MM-DD
	Nationality         string                       `json:"nationality"`           // NG
	Occupation          string                       `json:"occupation,omitempty"`
	Email               string                       `json:"email"`
	Phone               string                       `json:"phone"`                 // +2348133443000
	BankIDNumber        string                       `json:"bank_id_number,omitempty"` // BVN or similar
	SourceOfInflow      string                       `json:"source_of_inflow"`      // bank_statement, salary, business_income, etc.
	SourceOfInflowDocument *FileReference           `json:"source_of_inflow_document,omitempty"`
	Selfie              *FileReference              `json:"selfie,omitempty"`
	Identification      *AccountHolderIdentification `json:"identification,omitempty"`
	ProofOfAddress      *AccountHolderProofOfAddress `json:"proof_of_address,omitempty"`
	Address             *AccountHolderAddress       `json:"address,omitempty"`
	Employment          *AccountHolderEmployment    `json:"employment,omitempty"`
	Metadata            map[string]interface{}      `json:"metadata,omitempty"`
}

// AccountHolderCreateResponseData represents the response data from creating an account holder
type AccountHolderCreateResponseData struct {
	Reference string                 `json:"reference"` // e.g., "KPY-AH-CGAXuc6jZwDA8TJ"
	Email     string                 `json:"email"`
	Status    string                 `json:"status"` // pending, approved, rejected
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AccountHolderCreateResponse represents the response from creating an account holder
type AccountHolderCreateResponse struct {
	Status  bool                          `json:"status"`
	Message string                        `json:"message"`
	Data    *AccountHolderCreateResponseData `json:"data,omitempty"`
}

// AccountHolderDetailsDocuments represents document references in account holder details
type AccountHolderDetailsDocuments struct {
	IdentificationFront string `json:"identification_front,omitempty"` // Base64 encoded
	IdentificationBack  string `json:"identification_back,omitempty"`  // Base64 encoded
	ProofOfAddress      string `json:"proof_of_address,omitempty"`     // Base64 encoded
	Selfie              string `json:"selfie,omitempty"`               // Base64 encoded
	SourceOfInflow      string `json:"source_of_inflow,omitempty"`     // Base64 encoded
}

// AccountHolderDetailsData represents the detailed account holder information
type AccountHolderDetailsData struct {
	Reference      string                      `json:"reference"`
	AccountType    string                      `json:"account_type"`    // individual, business
	FirstName      string                      `json:"first_name"`
	LastName       string                      `json:"last_name"`
	Email          string                      `json:"email"`
	PhoneNumber    string                      `json:"phone_number"`
	Occupation     string                      `json:"occupation,omitempty"`
	Status         string                      `json:"status"`          // pending, approved, rejected
	Metadata       map[string]interface{}      `json:"metadata,omitempty"`
	DateCreated    string                      `json:"date_created"`    // ISO 8601
	Country        string                      `json:"country"`
	DateOfBirth     string                      `json:"date_of_birth,omitempty"` // ISO 8601
	Address         *AccountHolderAddress      `json:"address,omitempty"`
	Documents       *AccountHolderDetailsDocuments `json:"documents,omitempty"`
}

// AccountHolderDetailsResponse represents the response from getting account holder details
type AccountHolderDetailsResponse struct {
	Status  bool                      `json:"status"`
	Message string                    `json:"message"`
	Data    *AccountHolderDetailsData `json:"data,omitempty"`
}

// AccountHolderUpdateKYCRequest represents the request to update account holder KYC
type AccountHolderUpdateKYCRequest struct {
	FirstName           string                       `json:"first_name"`
	LastName            string                       `json:"last_name"`
	SourceOfInflow      string                       `json:"source_of_inflow"`
	SourceOfInflowDocument *FileReference           `json:"source_of_inflow_document,omitempty"`
	Selfie              *FileReference              `json:"selfie,omitempty"`
	Identification      *AccountHolderIdentification `json:"identification,omitempty"`
	ProofOfAddress      *AccountHolderProofOfAddress `json:"proof_of_address,omitempty"`
}

// AccountHolderUpdateKYCResponseData represents the response data from updating KYC
type AccountHolderUpdateKYCResponseData struct {
	Reference string `json:"reference"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Status    string `json:"status"` // pending
}

// AccountHolderUpdateKYCResponse represents the response from updating account holder KYC
type AccountHolderUpdateKYCResponse struct {
	Status  bool                           `json:"status"`
	Message string                         `json:"message"`
	Data    *AccountHolderUpdateKYCResponseData `json:"data,omitempty"`
}

// ============================================================================
// File Upload URL Types
// ============================================================================

// FileUploadURLRequest represents the request to generate a file upload URL
type FileUploadURLRequest struct {
	Reference   string `json:"reference"`    // Your unique reference for the file
	Purpose     string `json:"purpose"`      // kyc_document, proof_of_address, etc.
	ContentType string `json:"content_type"` // image/jpeg, application/pdf, image/png
}

// FileUploadURLResponseData represents the response data for file upload URL
type FileUploadURLResponseData struct {
	KorapayReference string `json:"korapay_reference"`  // e.g., "KPY-FILE-202508291414WUPh6CmvLAU052iYVynj14842"
	OwnerReference   string `json:"owner_reference"`    // Your reference
	Purpose          string `json:"purpose"`
	UploadURL        string `json:"upload_url"`         // Pre-signed S3 URL
	UploadURLExpires string `json:"upload_url_expires"` // ISO 8601
}

// FileUploadURLResponse represents the response from generating file upload URL
type FileUploadURLResponse struct {
	Status  bool                     `json:"status"`
	Message string                   `json:"message"`
	Data    *FileUploadURLResponseData `json:"data,omitempty"`
}
