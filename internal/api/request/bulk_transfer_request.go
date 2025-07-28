package request

import "payflow/internal/domain"

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
	FromAccountDetails *domain.AccountEnquiryData     `json:"from_account_details,omitempty"`
	ToAccountDetails   *domain.BeneficiaryEnquiryData `json:"to_account_details,omitempty"`
	BankCode           string                         `json:"bank_code,omitempty"`
}

// BulkTransferBatchRequest represents a batch of transfer requests
type BulkTransferBatchRequest struct {
	Transfers []BulkTransferRequest `json:"transfers" validate:"required,min=1,max=100"`
}
