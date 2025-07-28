package response

import "time"

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

// BulkTransferBatchResponse represents the response from a batch transfer operation
type BulkTransferBatchResponse struct {
	TotalTransfers      int                    `json:"total_transfers"`
	SuccessfulTransfers int                    `json:"successful_transfers"`
	FailedTransfers     int                    `json:"failed_transfers"`
	Transfers           []BulkTransferResponse `json:"transfers"`
	ProcessingTime      time.Duration          `json:"processing_time"`
}
