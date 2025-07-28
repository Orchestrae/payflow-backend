package response

import "time"

// AccountEnquiryResponse represents the response for account enquiry
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

// BeneficiaryEnquiryResponse represents the response for beneficiary enquiry
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

// BankListResponse represents the response for bank list
type BankListResponse struct {
	Status  string     `json:"status"`
	Message string     `json:"message"`
	Data    []BankData `json:"data"`
}

type BankData struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// TransferResponse represents the response for transfer
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

// TransferRecordResponse represents the response for transfer records
type TransferRecordResponse struct {
	ID              uint       `json:"id"`
	BusinessID      uint       `json:"business_id"`
	FromAccount     string     `json:"from_account"`
	FromClientId    string     `json:"from_client_id"`
	FromClient      string     `json:"from_client"`
	FromSavingsId   string     `json:"from_savings_id"`
	FromBvn         *string    `json:"from_bvn,omitempty"`
	ToClientId      string     `json:"to_client_id"`
	ToClient        string     `json:"to_client"`
	ToSavingsId     string     `json:"to_savings_id"`
	ToSession       *string    `json:"to_session,omitempty"`
	ToBvn           *string    `json:"to_bvn,omitempty"`
	ToAccount       string     `json:"to_account"`
	ToBank          string     `json:"to_bank"`
	Amount          string     `json:"amount"`
	Remark          string     `json:"remark"`
	TransferType    string     `json:"transfer_type"`
	Reference       string     `json:"reference"`
	TxnId           *string    `json:"txn_id,omitempty"`
	SessionId       *string    `json:"session_id,omitempty"`
	Status          string     `json:"status"`
	VFDStatus       string     `json:"vfd_status"`
	VFDMessage      string     `json:"vfd_message"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	ProcessingError *string    `json:"processing_error,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// TransferListResponse represents the response for listing transfers
type TransferListResponse struct {
	Transfers []TransferRecordResponse `json:"transfers"`
	Total     int                      `json:"total"`
	Page      int                      `json:"page"`
	Limit     int                      `json:"limit"`
}
