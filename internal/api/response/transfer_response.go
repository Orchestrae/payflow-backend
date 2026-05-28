package response

// SingleTransferResponse represents the response from a transfer operation
type SingleTransferResponse struct {
	Success        bool   `json:"success"`
	TransferID     uint   `json:"transfer_id,omitempty"`
	Reference      string `json:"reference"`
	TransactionID  string `json:"transaction_id,omitempty"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	Provider       string `json:"provider,omitempty"`
	Currency       string `json:"currency,omitempty"`
	Fee            string `json:"fee,omitempty"`
	ProcessingTime string `json:"processing_time"`
	Error          string `json:"error,omitempty"`
}

// BatchTransferResponse represents the response from a batch transfer operation
type BatchTransferResponse struct {
	TotalTransfers      int                      `json:"total_transfers"`
	SuccessfulTransfers int                      `json:"successful_transfers"`
	FailedTransfers     int                      `json:"failed_transfers"`
	Transfers           []SingleTransferResponse `json:"transfers"`
	ProcessingTime      string                   `json:"processing_time"`
}
