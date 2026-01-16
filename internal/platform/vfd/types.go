// internal/platform/vfd/types.go
package vfd

import "time"

// ============================================================================
// Authentication Types
// ============================================================================

// TokenRequest is the request body for generating a VFD access token.
type TokenRequest struct {
	ConsumerKey    string `json:"consumerKey"`
	ConsumerSecret string `json:"consumerSecret"`
	ValidityTime   string `json:"validityTime"` // "-1" for non-expiring token
}

// TokenResponseData holds the access token response from VFD.
type TokenResponseData struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ============================================================================
// Account Enquiry Types
// ============================================================================

// AccountEnquiryData holds account details from the enquiry endpoint.
type AccountEnquiryData struct {
	AccountNo          string `json:"accountNo"`
	AccountBalance     string `json:"accountBalance"`
	AccountId          string `json:"accountId"`
	Client             string `json:"client"`
	ClientId           string `json:"clientId"`
	SavingsProductName string `json:"savingsProductName"`
}

// ============================================================================
// Beneficiary Enquiry Types
// ============================================================================

// BeneficiaryAccount holds the account details within beneficiary response.
type BeneficiaryAccount struct {
	Number string `json:"number"`
	ID     string `json:"id"`
}

// BeneficiaryEnquiryData holds beneficiary details from the enquiry endpoint.
type BeneficiaryEnquiryData struct {
	Name     string             `json:"name"`
	ClientId string             `json:"clientId"`
	BVN      string             `json:"bvn"`
	Account  BeneficiaryAccount `json:"account"`
	Status   string             `json:"status"`
	Currency string             `json:"currency"`
	Bank     string             `json:"bank"`
}

// ============================================================================
// Bank List Types
// ============================================================================

// BankData holds bank information.
type BankData struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// ============================================================================
// Transfer Types
// ============================================================================

// TransferRequest is the request body for the transfer endpoint.
type TransferRequest struct {
	FromAccount           string `json:"fromAccount"`
	UniqueSenderAccountId string `json:"uniqueSenderAccountId,omitempty"`
	FromClientId          string `json:"fromClientId"`
	FromClient            string `json:"fromClient"`
	FromSavingsId         string `json:"fromSavingsId"`
	FromBvn               string `json:"fromBvn,omitempty"`
	ToClientId            string `json:"toClientId,omitempty"`  // Mandatory for intra
	ToClient              string `json:"toClient"`
	ToSavingsId           string `json:"toSavingsId,omitempty"` // Mandatory for intra
	ToSession             string `json:"toSession,omitempty"`   // Mandatory for inter
	ToBvn                 string `json:"toBvn,omitempty"`
	ToAccount             string `json:"toAccount"`
	ToBank                string `json:"toBank"`
	Signature             string `json:"signature"`
	Amount                string `json:"amount"`
	Remark                string `json:"remark"`
	TransferType          string `json:"transferType"` // "intra" or "inter"
	Reference             string `json:"reference"`
}

// TransferResponseData holds the transfer response data.
type TransferResponseData struct {
	TxnId     string `json:"txnId"`
	SessionId string `json:"sessionId,omitempty"` // Only for inter-bank
	Reference string `json:"reference,omitempty"`
}

// ============================================================================
// Transaction Status Query (TSQ) Types
// ============================================================================

// TransactionStatusData holds the transaction status from TSQ endpoint.
type TransactionStatusData struct {
	TxnId             string `json:"TxnId"`
	Amount            string `json:"amount"`
	AccountNo         string `json:"accountNo"`
	FromAccountNo     string `json:"fromAccountNo"`
	TransactionStatus string `json:"transactionStatus"`
	TransactionDate   string `json:"transactionDate"`
	ToBank            string `json:"toBank"`
	FromBank          string `json:"fromBank"`
	SessionId         string `json:"sessionId"`
	BankTransactionId string `json:"bankTransactionId"`
	TransactionType   string `json:"transactionType"`
}

// ============================================================================
// Generic VFD Response Wrapper
// ============================================================================

// VFDResponse is a generic wrapper for all VFD API responses.
type VFDResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ============================================================================
// Corporate Account Creation Types (existing)
// ============================================================================

// CreateCorporateRequest is the request body for the /corporateclient/create endpoint.
type CreateCorporateRequest struct {
	RCNumber          string `json:"rcNumber,omitempty"`
	CompanyName       string `json:"companyName,omitempty"`
	IncorporationDate string `json:"incorporationDate,omitempty"`
	BVN               string `json:"bvn,omitempty"`
	PreviousAccountNo string `json:"previousAccountNo,omitempty"`
}

// CorporateAccountData holds the details of the successfully created account.
type CorporateAccountData struct {
	AccountNo   string `json:"accountNo"`
	AccountName string `json:"accountName"`
}

// ============================================================================
// Clean Domain-like Structs for Service Layer
// ============================================================================

// NewAccountDetails is a clean struct used to pass new account information to the service.
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
