package request

// CreateVirtualAccountRequest represents the request to create a virtual account
type CreateVirtualAccountRequest struct {
	AccountName     string `json:"account_name" validate:"required"`
	AccountReference string `json:"account_reference,omitempty"` // Optional - will be auto-generated if not provided
	CustomerName    string `json:"customer_name" validate:"required"`
	CustomerEmail   string `json:"customer_email,omitempty"`
	BVN             string `json:"bvn" validate:"required,len=11"`
	NIN             string `json:"nin,omitempty"` // Optional
	BankCode        string `json:"bank_code,omitempty"` // Optional - provider may assign
	Permanent       bool   `json:"permanent,omitempty"` // Defaults to true
}

// SandboxCreditRequest represents the request to credit a virtual account in sandbox (testing only)
type SandboxCreditRequest struct {
	AccountNumber string `json:"account_number" validate:"required"`
	Amount        int    `json:"amount" validate:"required,min=1"` // Amount in main currency unit (e.g., NGN)
	Currency      string `json:"currency,omitempty"`                // Defaults to NGN
}

// ============================================================================
// Account Holder / KYC Request Types
// ============================================================================

// FileReferenceRequest represents a file reference for document uploads
type FileReferenceRequest struct {
	Reference string `json:"reference" validate:"required"`
}

// AccountHolderIdentificationRequest represents identification document details
type AccountHolderIdentificationRequest struct {
	Type         string                 `json:"type" validate:"required"` // passport, national_id, driver_license, etc.
	Number       string                 `json:"number" validate:"required"`
	DocumentFront *FileReferenceRequest `json:"document_front,omitempty"`
	DocumentBack  *FileReferenceRequest `json:"document_back,omitempty"`
	IssuedDate   string                 `json:"issued_date,omitempty"` // YYYY-MM-DD
	ExpiryDate   string                 `json:"expiry_date,omitempty"` // YYYY-MM-DD
	Country      string                 `json:"country,omitempty"`     // NG, etc.
}

// AccountHolderProofOfAddressRequest represents proof of address document
type AccountHolderProofOfAddressRequest struct {
	Type     string                 `json:"type" validate:"required"` // bank_statement, utility_bill, etc.
	Document *FileReferenceRequest  `json:"document,omitempty"`
}

// AccountHolderAddressRequest represents physical address details
type AccountHolderAddressRequest struct {
	Country string `json:"country" validate:"required"` // NG
	Zip     string `json:"zip,omitempty"`
	Address string `json:"address" validate:"required"` // Street address
	State   string `json:"state" validate:"required"`
	City    string `json:"city" validate:"required"`
}

// AccountHolderEmploymentRequest represents employment details
type AccountHolderEmploymentRequest struct {
	Status      string `json:"status" validate:"required"` // employer, employee, self_employed, unemployed, student
	Employer    string `json:"employer,omitempty"`
	Description string `json:"description,omitempty"`
}

// CreateAccountHolderRequest represents the request to create an account holder (KYC onboarding)
type CreateAccountHolderRequest struct {
	FirstName             string                              `json:"first_name" validate:"required"`
	LastName              string                              `json:"last_name" validate:"required"`
	UseCase               string                              `json:"use_case" validate:"required"` // Personal, Business
	Type                  string                              `json:"type" validate:"required"`     // individual, business
	DateOfBirth           string                              `json:"date_of_birth" validate:"required"` // YYYY-MM-DD
	Nationality           string                              `json:"nationality" validate:"required"`   // NG
	Occupation            string                              `json:"occupation,omitempty"`
	Email                 string                              `json:"email" validate:"required,email"`
	Phone                 string                              `json:"phone" validate:"required"` // +2348133443000
	BankIDNumber          string                              `json:"bank_id_number,omitempty"` // BVN or similar
	SourceOfInflow        string                              `json:"source_of_inflow" validate:"required"` // bank_statement, salary, business_income, etc.
	SourceOfInflowDocument *FileReferenceRequest              `json:"source_of_inflow_document,omitempty"`
	Selfie                *FileReferenceRequest               `json:"selfie,omitempty"`
	Identification        *AccountHolderIdentificationRequest `json:"identification,omitempty"`
	ProofOfAddress        *AccountHolderProofOfAddressRequest `json:"proof_of_address,omitempty"`
	Address               *AccountHolderAddressRequest        `json:"address,omitempty"`
	Employment            *AccountHolderEmploymentRequest     `json:"employment,omitempty"`
	Metadata              map[string]interface{}              `json:"metadata,omitempty"`
}

// UpdateAccountHolderKYCRequest represents the request to update account holder KYC information
type UpdateAccountHolderKYCRequest struct {
	FirstName             string                              `json:"first_name" validate:"required"`
	LastName              string                              `json:"last_name" validate:"required"`
	SourceOfInflow        string                              `json:"source_of_inflow" validate:"required"`
	SourceOfInflowDocument *FileReferenceRequest              `json:"source_of_inflow_document,omitempty"`
	Selfie                *FileReferenceRequest               `json:"selfie,omitempty"`
	Identification        *AccountHolderIdentificationRequest `json:"identification,omitempty"`
	ProofOfAddress        *AccountHolderProofOfAddressRequest `json:"proof_of_address,omitempty"`
}

// GenerateFileUploadURLRequest represents the request to generate a file upload URL
type GenerateFileUploadURLRequest struct {
	Reference   string `json:"reference" validate:"required"`   // Your unique reference for the file
	Purpose     string `json:"purpose" validate:"required"`     // kyc_document, proof_of_address, etc.
	ContentType string `json:"content_type" validate:"required"` // image/jpeg, application/pdf, image/png
}
