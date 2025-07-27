// internal/platform/vfd/types.go
package vfd

import "time"

// tokenRequest is the request body for generating a VFD access token.
type tokenRequest struct {
	// ConsumerKey is the wallet's unique consumer key from the VBaaS portal.
	ConsumerKey string `json:"consumerKey"`
	// ConsumerSecret is the wallet's unique consumer secret from the VBaaS portal.
	ConsumerSecret string `json:"consumerSecret"`
	// ValidityTime sets the token's validity in minutes. A value of -1 creates a non-expiring token.
	ValidityTime string `json:"validityTime"`
}

// tokenResponseData holds the actual access token information.
type tokenResponseData struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// --- Corporate Account Creation Structs ---

// CreateCorporateRequest is the request body for the /corporateclient/create endpoint.
// The `omitempty` tag is used strategically so we can use the same struct for both new and duplicate account creation.
type CreateCorporateRequest struct {
	// RCNumber is the company's registration certificate number. Mandatory only for new account creation.
	RCNumber string `json:"rcNumber,omitempty"`
	// CompanyName is the legal name of the company. Mandatory only for new account creation.
	CompanyName string `json:"companyName,omitempty"`
	// IncorporationDate is the date the company was incorporated. Must be in "02 January 2006" format. Mandatory only for new account creation.
	IncorporationDate string `json:"incorporationDate,omitempty"`
	// BVN is the Bank Verification Number of one of the company's directors. Mandatory only for new account creation.
	BVN string `json:"bvn,omitempty"`
	// PreviousAccountNo is used to create a duplicate account for an existing client. If provided, all other fields should be empty.
	PreviousAccountNo string `json:"previousAccountNo,omitempty"`
}

// corporateAccountData holds the details of the successfully created account.
type corporateAccountData struct {
	AccountNo   string `json:"accountNo"`
	AccountName string `json:"accountName"`
}

// --- Generic VFD Response Wrapper ---

// vfdResponse is a generic wrapper for all VFD API responses.
// The `Data` field is an `interface{}` to handle different data structures (token vs. account).
type vfdResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// --- Clean, Domain-like Structs for Service Layer ---

// NewAccountDetails is a clean struct used to pass new account information to the service.
// This decouples the service user from VFD's specific field names.
type NewAccountDetails struct {
	RCNumber          string
	CompanyName       string
	IncorporationDate time.Time
	DirectorBVN       string
}

// CorporateAccount is the clean, final output from the service.
type CorporateAccount struct {
	AccountNumber string
	AccountName   string
}
