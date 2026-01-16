package domain

import "time"

// VFDTransferService represents the transfer service domain models

// AccountEnquiryResponse represents the response from account enquiry API
type AccountEnquiryResponse struct {
	Status  string              `json:"status"`
	Message string              `json:"message"`
	Data    *AccountEnquiryData `json:"data,omitempty"`
}

type AccountEnquiryData struct {
	AccountNo          string `json:"accountNo"`
	AccountBalance     string `json:"accountBalance"`
	AccountId          string `json:"accountId"`
	Client             string `json:"client"`
	ClientId           string `json:"clientId"`
	SavingsProductName string `json:"savingsProductName"`
}

// BeneficiaryEnquiryResponse represents the response from beneficiary enquiry API
type BeneficiaryEnquiryResponse struct {
	Status  string                  `json:"status"`
	Message string                  `json:"message"`
	Data    *BeneficiaryEnquiryData `json:"data,omitempty"`
}

type BeneficiaryEnquiryData struct {
	Name     string `json:"name"`
	ClientId string `json:"clientId"`
	BVN      string `json:"bvn"`
	Account  struct {
		Number string `json:"number"`
		ID     string `json:"id"`
	} `json:"account"`
	Status   string `json:"status"`
	Currency string `json:"currency"`
	Bank     string `json:"bank"`
}

// BankListResponse represents the response from bank list API
type BankListResponse struct {
	Status  string     `json:"status"`
	Message string     `json:"message"`
	Data    []BankData `json:"data"`
}

type BankData struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// TransferRequest represents the request for transfer API
type TransferRequest struct {
	FromAccount           string `json:"fromAccount" validate:"required"`
	UniqueSenderAccountId string `json:"uniqueSenderAccountId"`
	FromClientId          string `json:"fromClientId" validate:"required"`
	FromClient            string `json:"fromClient" validate:"required"`
	FromSavingsId         string `json:"fromSavingsId" validate:"required"`
	FromBvn               string `json:"fromBvn"`
	ToClientId            string `json:"toClientId" validate:"required"`
	ToClient              string `json:"toClient" validate:"required"`
	ToSavingsId           string `json:"toSavingsId" validate:"required"`
	ToSession             string `json:"toSession"`
	ToBvn                 string `json:"toBvn"`
	ToAccount             string `json:"toAccount" validate:"required"`
	ToBank                string `json:"toBank" validate:"required"`
	Signature             string `json:"signature" validate:"required"`
	Amount                string `json:"amount" validate:"required"`
	Remark                string `json:"remark" validate:"required"`
	TransferType          string `json:"transferType" validate:"required,oneof=intra inter"`
	Reference             string `json:"reference" validate:"required"`
}

// TransferResponse represents the response from transfer API
type TransferResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Data    *TransferData `json:"data,omitempty"`
}

type TransferData struct {
	TxnId     string `json:"txnId"`
	SessionId string `json:"sessionId,omitempty"`
	Reference string `json:"reference"`
}

// TransferRecord represents a transfer record in our database
type TransferRecord struct {
	Model
	BusinessID      uint    `gorm:"index"`
	FromAccount     string  `gorm:"size:20;index"`
	FromClientId    string  `gorm:"size:20"`
	FromClient      string  `gorm:"size:255"`
	FromSavingsId   string  `gorm:"size:20"`
	FromBvn         *string `gorm:"size:11"`
	ToClientId      string  `gorm:"size:20"`
	ToClient        string  `gorm:"size:255"`
	ToSavingsId     string  `gorm:"size:20"`
	ToSession       *string `gorm:"size:50"`
	ToBvn           *string `gorm:"size:11"`
	ToAccount       string  `gorm:"size:20;index"`
	ToBank          string  `gorm:"size:10"`
	Amount          string  `gorm:"size:20"`
	Remark          string  `gorm:"size:500"`
	TransferType    string  `gorm:"size:10"`
	Reference       string  `gorm:"size:100;uniqueIndex"`
	TxnId           *string `gorm:"size:100"`
	SessionId       *string `gorm:"size:50"`
	Status          string  `gorm:"size:20;default:'pending'"` // pending, success, failed
	VFDStatus       string  `gorm:"size:10"`                   // VFD response status
	VFDMessage      string  `gorm:"size:255"`                  // VFD response message
	ProcessedAt     *time.Time
	ProcessingError *string `gorm:"size:1000"`

	// Relationships
	Business *Business `gorm:"-"`
}

// TransferStatus represents the status of a transfer
type TransferStatus string

const (
	TransferStatusPending TransferStatus = "pending"
	TransferStatusSuccess TransferStatus = "success"
	TransferStatusFailed  TransferStatus = "failed"
)

// TransferType represents the type of transfer
type TransferType string

const (
	TransferTypeIntra TransferType = "intra" // VFD to VFD
	TransferTypeInter TransferType = "inter" // VFD to other banks
)

// VFD API Error Responses
type VFDErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    *struct {
		TxnId string `json:"txnId"`
	} `json:"data,omitempty"`
}

// TransactionStatusResponse represents the response from transaction status query
type TransactionStatusResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    *TransactionStatusData `json:"data,omitempty"`
}

// TransactionStatusData holds the transaction status details
type TransactionStatusData struct {
	TxnId             string `json:"TxnId"`
	Amount            string `json:"amount"`
	AccountNo         string `json:"accountNo"`
	FromAccountNo     string `json:"fromAccountNo"`
	TransactionStatus string `json:"transactionStatus"` // "00" = success, "99" = failed
	TransactionDate   string `json:"transactionDate"`
	ToBank            string `json:"toBank"`
	FromBank          string `json:"fromBank"`
	SessionId         string `json:"sessionId"`
	BankTransactionId string `json:"bankTransactionId"`
	TransactionType   string `json:"transactionType"` // OUTFLOW, INFLOW
}
