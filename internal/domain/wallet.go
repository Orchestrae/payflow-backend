package domain

import "time"

// ============================================================================
// Wallet & Virtual Account Domain Models (Provider-Agnostic)
// ============================================================================

// BusinessWallet represents a business's wallet with balance and virtual account details.
// This is the provider-agnostic representation stored in our database.
type BusinessWallet struct {
	Model

	// Business relationship
	BusinessID uint `gorm:"index"`

	// Balance tracking (in smallest currency unit, e.g., kobo for NGN)
	Balance      int64     `gorm:"default:0"` // Available balance
	LockedBalance int64    `gorm:"default:0"` // Balance locked for pending transfers
	Currency     string    `gorm:"size:10;default:'NGN'"`
	BalanceUpdatedAt *time.Time // When balance was last synced from provider

	// Virtual Account Details (stored from provider after creation)
	VirtualAccountNumber string `gorm:"size:20;uniqueIndex"`
	VirtualAccountBankCode string `gorm:"size:10"`
	VirtualAccountBankName string `gorm:"size:255"`
	VirtualAccountReference string `gorm:"size:100;uniqueIndex"` // Provider's account reference
	VirtualAccountUniqueID string `gorm:"size:100"` // Provider's unique ID (e.g., KPY-VA-xxx)
	VirtualAccountStatus string `gorm:"size:20;default:'active'"` // active, suspended, inactive

	// Provider information
	Provider ProviderName `gorm:"size:20"` // korapay, vfd, etc.
	ProviderMetadata string `gorm:"type:jsonb"` // JSON for provider-specific fields

	// Relationships
	Business *Business `gorm:"-"`
}

// TableName specifies the table name for GORM
func (BusinessWallet) TableName() string {
	return "business_wallets"
}

// WalletTransaction represents a transaction in the wallet (deposit, withdrawal, fee, refund).
// This provides a complete audit trail of all wallet activity.
type WalletTransactionType string

const (
	WalletTransactionDeposit    WalletTransactionType = "deposit"
	WalletTransactionWithdrawal WalletTransactionType = "withdrawal"
	WalletTransactionFee        WalletTransactionType = "fee"
	WalletTransactionRefund     WalletTransactionType = "refund"
)

type WalletTransaction struct {
	Model

	// Business relationship
	BusinessID uint `gorm:"index"`

	// Transaction details
	TransactionType WalletTransactionType `gorm:"size:20"`
	Amount          int64                 `gorm:""` // Always positive (use type to indicate direction)
	BalanceBefore   int64                 `gorm:""` // Balance before this transaction
	BalanceAfter    int64                 `gorm:""` // Balance after this transaction
	Currency        string                `gorm:"size:10;default:'NGN'"`

	// Reference tracking
	Reference        string `gorm:"size:100;uniqueIndex"` // Our internal reference
	ProviderReference string `gorm:"size:100"`            // Provider's transaction reference (e.g., KPY-PAY-xxx)
	Description      string `gorm:"size:500"`             // Human-readable description

	// Link to related transfer (if this is a withdrawal)
	TransferID *uint `gorm:"index"` // Nullable - only set for withdrawals

	// Status
	Status string `gorm:"size:20;default:'completed'"` // completed, pending, failed

	// Timestamps
	ProcessedAt *time.Time

	// Relationships
	Business *Business `gorm:"-"`
	Transfer *Transfer `gorm:"-"`
}

// TableName specifies the table name for GORM
func (WalletTransaction) TableName() string {
	return "wallet_transactions"
}

// ============================================================================
// Virtual Account Request/Response (Provider-Agnostic)
// ============================================================================

// CreateVirtualAccountRequest represents a provider-agnostic request to create a virtual account.
type CreateVirtualAccountRequest struct {
	// Business details
	BusinessID    uint   `json:"-"`
	AccountName   string `json:"account_name" validate:"required"`
	AccountReference string `json:"account_reference"` // Auto-generated if not provided

	// Customer details (for KYC)
	CustomerName  string `json:"customer_name" validate:"required"`
	CustomerEmail string `json:"customer_email,omitempty"`

	// KYC details
	BVN string `json:"bvn" validate:"required,len=11"`
	NIN string `json:"nin,omitempty"` // Optional

	// Provider-specific options (will be mapped by provider implementation)
	BankCode string `json:"bank_code,omitempty"` // Optional - provider may assign
	Permanent bool  `json:"permanent,omitempty"` // true for permanent accounts (Kora default)
}

// VirtualAccountResult represents the unified result from any virtual account provider.
type VirtualAccountResult struct {
	Success                bool         `json:"success"`
	Provider              ProviderName `json:"provider"`
	AccountNumber         string       `json:"account_number"`
	AccountName           string       `json:"account_name"`
	BankCode              string       `json:"bank_code"`
	BankName              string       `json:"bank_name"`
	AccountReference      string       `json:"account_reference"` // Our reference
	UniqueID              string       `json:"unique_id"`         // Provider's unique ID (e.g., KPY-VA-xxx)
	AccountStatus         string       `json:"account_status"`    // active, suspended, inactive
	Currency              string       `json:"currency"`
	CreatedAt             time.Time    `json:"created_at"`
	Message               string       `json:"message,omitempty"`
}

// VirtualAccountBalanceResult represents balance information from a provider.
type VirtualAccountBalanceResult struct {
	Success              bool         `json:"success"`
	Provider             ProviderName `json:"provider"`
	AccountNumber        string       `json:"account_number"`
	AccountReference     string       `json:"account_reference"`
	Balance              int64        `json:"balance"` // In smallest currency unit
	Currency             string       `json:"currency"`
	AvailableBalance     int64        `json:"available_balance"` // Balance minus locked amount
	LastUpdated          time.Time    `json:"last_updated"`
}

// VirtualAccountTransaction represents a transaction on a virtual account (from provider).
type VirtualAccountTransaction struct {
	Reference         string    `json:"reference"` // Provider's transaction reference
	Status            string    `json:"status"`    // success, pending, failed
	Amount            int64     `json:"amount"`    // In smallest currency unit
	Fee               int64     `json:"fee,omitempty"`
	Currency          string    `json:"currency"`
	Description       string    `json:"description,omitempty"`
	ProcessedAt       time.Time `json:"processed_at"`
	PayerBankAccount  *PayerBankAccount `json:"payer_bank_account,omitempty"` // For deposits
}

// PayerBankAccount represents payer details for a deposit transaction.
type PayerBankAccount struct {
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	BankName      string `json:"bank_name"`
}

// VirtualAccountTransactionsResult represents a list of transactions with pagination.
type VirtualAccountTransactionsResult struct {
	Success             bool                      `json:"success"`
	Provider            ProviderName              `json:"provider"`
	AccountNumber       string                    `json:"account_number"`
	TotalAmountReceived int64                     `json:"total_amount_received"` // In smallest currency unit
	Currency            string                    `json:"currency"`
	Transactions        []VirtualAccountTransaction `json:"transactions"`
	Pagination          *PaginationInfo           `json:"pagination,omitempty"`
}

// PaginationInfo represents pagination metadata.
type PaginationInfo struct {
	Page       int `json:"page"`
	Total      int `json:"total"`
	PageCount  int `json:"pageCount"`
	TotalPages int `json:"totalPages"`
}

// DepositNotification represents a deposit notification (used in webhooks).
type DepositNotification struct {
	Provider          ProviderName
	Reference         string // Provider's transaction reference (e.g., KPY-PAY-xxx)
	AccountNumber     string
	AccountReference  string
	Amount            int64 // In smallest currency unit
	Currency          string
	Status            string // success, pending, failed
	Description       string
	ProcessedAt       time.Time
	PayerBankAccount  *PayerBankAccount
}

// ============================================================================
// Account Holder / KYC Domain Models (Provider-Agnostic)
// ============================================================================

// FileReference represents a file reference for document uploads
type FileReference struct {
	Reference string `json:"reference"`
}

// AccountHolderIdentification represents identification document details
type AccountHolderIdentification struct {
	Type         string         `json:"type"` // passport, national_id, driver_license, etc.
	Number       string         `json:"number"`
	DocumentFront *FileReference `json:"document_front,omitempty"`
	DocumentBack  *FileReference `json:"document_back,omitempty"`
	IssuedDate   string         `json:"issued_date,omitempty"` // YYYY-MM-DD
	ExpiryDate   string         `json:"expiry_date,omitempty"` // YYYY-MM-DD
	Country      string         `json:"country,omitempty"`    // NG, etc.
}

// AccountHolderProofOfAddress represents proof of address document
type AccountHolderProofOfAddress struct {
	Type     string         `json:"type"` // bank_statement, utility_bill, etc.
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
	Status      string `json:"status"` // employer, employee, self_employed, unemployed, student
	Employer    string `json:"employer,omitempty"`
	Description string `json:"description,omitempty"`
}

// CreateAccountHolderRequest represents a provider-agnostic request to create an account holder
type CreateAccountHolderRequest struct {
	FirstName             string                       `json:"first_name"`
	LastName              string                       `json:"last_name"`
	UseCase               string                       `json:"use_case"` // Personal, Business
	Type                  string                       `json:"type"`     // individual, business
	DateOfBirth           string                       `json:"date_of_birth"` // YYYY-MM-DD
	Nationality           string                       `json:"nationality"`   // NG
	Occupation            string                       `json:"occupation,omitempty"`
	Email                 string                       `json:"email"`
	Phone                 string                       `json:"phone"` // +2348133443000
	BankIDNumber          string                       `json:"bank_id_number,omitempty"` // BVN or similar
	SourceOfInflow        string                       `json:"source_of_inflow"` // bank_statement, salary, business_income, etc.
	SourceOfInflowDocument *FileReference              `json:"source_of_inflow_document,omitempty"`
	Selfie                *FileReference               `json:"selfie,omitempty"`
	Identification        *AccountHolderIdentification `json:"identification,omitempty"`
	ProofOfAddress        *AccountHolderProofOfAddress `json:"proof_of_address,omitempty"`
	Address               *AccountHolderAddress        `json:"address,omitempty"`
	Employment            *AccountHolderEmployment     `json:"employment,omitempty"`
	Metadata              map[string]interface{}      `json:"metadata,omitempty"`
}

// AccountHolderResult represents the result of creating an account holder
type AccountHolderResult struct {
	Reference string                 `json:"reference"` // Provider's reference (e.g., "KPY-AH-xxx")
	Email     string                 `json:"email"`
	Status    string                 `json:"status"` // pending, approved, rejected
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AccountHolderDetails represents detailed account holder information
type AccountHolderDetails struct {
	Reference      string                      `json:"reference"`
	AccountType    string                      `json:"account_type"` // individual, business
	FirstName      string                      `json:"first_name"`
	LastName       string                      `json:"last_name"`
	Email          string                      `json:"email"`
	PhoneNumber    string                      `json:"phone_number"`
	Occupation     string                      `json:"occupation,omitempty"`
	Status         string                      `json:"status"` // pending, approved, rejected
	Metadata       map[string]interface{}      `json:"metadata,omitempty"`
	DateCreated    time.Time                   `json:"date_created"`
	Country        string                      `json:"country"`
	DateOfBirth    *time.Time                  `json:"date_of_birth,omitempty"`
	Address        *AccountHolderAddress       `json:"address,omitempty"`
	Documents      *AccountHolderDocuments     `json:"documents,omitempty"`
}

// AccountHolderDocuments represents document references in account holder details
type AccountHolderDocuments struct {
	IdentificationFront string `json:"identification_front,omitempty"` // Base64 encoded or URL
	IdentificationBack  string `json:"identification_back,omitempty"`
	ProofOfAddress      string `json:"proof_of_address,omitempty"`
	Selfie              string `json:"selfie,omitempty"`
	SourceOfInflow      string `json:"source_of_inflow,omitempty"`
}

// UpdateAccountHolderKYCRequest represents a provider-agnostic request to update account holder KYC
type UpdateAccountHolderKYCRequest struct {
	FirstName             string                       `json:"first_name"`
	LastName              string                       `json:"last_name"`
	SourceOfInflow        string                       `json:"source_of_inflow"`
	SourceOfInflowDocument *FileReference              `json:"source_of_inflow_document,omitempty"`
	Selfie                *FileReference               `json:"selfie,omitempty"`
	Identification        *AccountHolderIdentification `json:"identification,omitempty"`
	ProofOfAddress        *AccountHolderProofOfAddress `json:"proof_of_address,omitempty"`
}

// UpdateAccountHolderKYCResult represents the result of updating account holder KYC
type UpdateAccountHolderKYCResult struct {
	Reference string `json:"reference"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Status    string `json:"status"` // pending
}

// FileUploadURLResult represents the result of generating a file upload URL
type FileUploadURLResult struct {
	KorapayReference string    `json:"korapay_reference"`  // Provider's file reference
	OwnerReference   string    `json:"owner_reference"`     // Your reference
	Purpose          string    `json:"purpose"`
	UploadURL        string    `json:"upload_url"`          // Pre-signed S3 URL
	UploadURLExpires time.Time `json:"upload_url_expires"`  // Expiration time
}

// GenerateFileUploadURLRequest represents a provider-agnostic request to generate a file upload URL
type GenerateFileUploadURLRequest struct {
	Reference   string `json:"reference"`   // Your unique reference for the file
	Purpose     string `json:"purpose"`     // kyc_document, proof_of_address, etc.
	ContentType string `json:"content_type"` // image/jpeg, application/pdf, image/png
}
