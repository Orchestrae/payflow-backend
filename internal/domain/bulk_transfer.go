package domain

import "time"

// ============================================================================
// Legacy VFD-specific types (deprecated - use provider-agnostic types in transfer.go)
// ============================================================================

// LegacyBulkTransferRequest represents a single transfer request for the legacy bulk transfer service
// Deprecated: Use SingleTransferRequest instead
type LegacyBulkTransferRequest struct {
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

// LegacyBulkTransferResponse represents the response from a legacy bulk transfer operation
// Deprecated: Use SingleTransferResponse instead
type LegacyBulkTransferResponse struct {
	Success            bool                    `json:"success"`
	TransferID         uint                    `json:"transfer_id"`
	Reference          string                  `json:"reference"`
	VFDResponse        *TransferResponse       `json:"vfd_response,omitempty"`
	Error              string                  `json:"error,omitempty"`
	ProcessingTime     time.Duration           `json:"processing_time"`
	FromAccountDetails *AccountEnquiryData     `json:"from_account_details,omitempty"`
	ToAccountDetails   *BeneficiaryEnquiryData `json:"to_account_details,omitempty"`
}

// LegacyBulkTransferBatchRequest represents a batch of transfer requests
// Deprecated: Use BulkTransferRequest instead
type LegacyBulkTransferBatchRequest struct {
	Transfers []LegacyBulkTransferRequest `json:"transfers" validate:"required,min=1,max=100"`
}

// LegacyBulkTransferBatchResponse represents the response from a batch transfer operation
// Deprecated: Use BulkTransferResponse instead
type LegacyBulkTransferBatchResponse struct {
	TotalTransfers      int                          `json:"total_transfers"`
	SuccessfulTransfers int                          `json:"successful_transfers"`
	FailedTransfers     int                          `json:"failed_transfers"`
	Transfers           []LegacyBulkTransferResponse `json:"transfers"`
	ProcessingTime      time.Duration                `json:"processing_time"`
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
