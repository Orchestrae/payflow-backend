package domain

import "time"

// BulkTransferRequest represents a single transfer request for the bulk transfer service
type BulkTransferRequest struct {
	FromAccountNumber string `json:"from_account_number" validate:"required"`
	ToAccountNumber   string `json:"to_account_number" validate:"required"`
	ToBankCode        string `json:"to_bank_code" validate:"required"`
	Amount            string `json:"amount" validate:"required"`
	Remark            string `json:"remark" validate:"required"`
	TransferType      string `json:"transfer_type" validate:"required,oneof=intra inter"`
	Reference         string `json:"reference" validate:"required"`
	// Optional fields that can be provided to skip API calls
	FromAccountDetails *AccountEnquiryData     `json:"from_account_details,omitempty"`
	ToAccountDetails   *BeneficiaryEnquiryData `json:"to_account_details,omitempty"`
	BankCode           string                  `json:"bank_code,omitempty"`
}

// BulkTransferResponse represents the response from a bulk transfer operation
type BulkTransferResponse struct {
	Success            bool                    `json:"success"`
	TransferID         uint                    `json:"transfer_id"`
	Reference          string                  `json:"reference"`
	VFDResponse        *TransferResponse       `json:"vfd_response,omitempty"`
	Error              string                  `json:"error,omitempty"`
	ProcessingTime     time.Duration           `json:"processing_time"`
	FromAccountDetails *AccountEnquiryData     `json:"from_account_details,omitempty"`
	ToAccountDetails   *BeneficiaryEnquiryData `json:"to_account_details,omitempty"`
}

// BulkTransferBatchRequest represents a batch of transfer requests
type BulkTransferBatchRequest struct {
	Transfers []BulkTransferRequest `json:"transfers" validate:"required,min=1,max=100"`
}

// BulkTransferBatchResponse represents the response from a batch transfer operation
type BulkTransferBatchResponse struct {
	TotalTransfers      int                    `json:"total_transfers"`
	SuccessfulTransfers int                    `json:"successful_transfers"`
	FailedTransfers     int                    `json:"failed_transfers"`
	Transfers           []BulkTransferResponse `json:"transfers"`
	ProcessingTime      time.Duration          `json:"processing_time"`
}

// TransferFlowData represents the complete data needed for a transfer
type TransferFlowData struct {
	FromAccountDetails *AccountEnquiryData
	ToAccountDetails   *BeneficiaryEnquiryData
	TransferRequest    *TransferRequest
	BusinessID         uint
}

// TransferFlowResult represents the result of a transfer flow
type TransferFlowResult struct {
	Success        bool
	TransferID     uint
	VFDResponse    *TransferResponse
	Error          error
	ProcessingTime time.Duration
}
