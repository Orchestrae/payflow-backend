package request

// VFDTransferRequest represents the request for transfer API
type VFDTransferRequest struct {
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

// AccountEnquiryRequest represents the request for account enquiry
type AccountEnquiryRequest struct {
	AccountNumber string `json:"accountNumber" validate:"omitempty"`
}

// BeneficiaryEnquiryRequest represents the request for beneficiary enquiry
type BeneficiaryEnquiryRequest struct {
	AccountNo    string `json:"accountNo" validate:"required"`
	Bank         string `json:"bank" validate:"required"`
	TransferType string `json:"transfer_type" validate:"required,oneof=intra inter"`
}
