package korapay

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/service/provider"
)

// korapayTransferProvider implements the TransferProvider interface for Korapay.
// This is in the korapay package to avoid import cycles.
type korapayTransferProvider struct {
	client *Client
}

// NewTransferProvider creates a new Korapay transfer provider.
func NewTransferProvider(client *Client) provider.TransferProvider {
	return &korapayTransferProvider{
		client: client,
	}
}

// Name returns the provider identifier.
func (p *korapayTransferProvider) Name() string {
	return "korapay"
}

// AccountEnquiry is not supported by Korapay API.
func (p *korapayTransferProvider) AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	return nil, fmt.Errorf("account enquiry not supported by korapay provider")
}

// BeneficiaryEnquiry is not supported by Korapay API.
func (p *korapayTransferProvider) BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	return nil, fmt.Errorf("beneficiary enquiry not supported by korapay provider")
}

// GetBankList is not supported by Korapay API.
func (p *korapayTransferProvider) GetBankList(ctx context.Context) (*domain.BankListResponse, error) {
	return nil, fmt.Errorf("bank list not supported by korapay provider")
}

// InitiateTransfer implements the TransferProvider interface by mapping domain.TransferRequest
// to Korapay's single disbursement API format.
func (p *korapayTransferProvider) InitiateTransfer(ctx context.Context, businessID uint, req *domain.TransferRequest) (*domain.TransferResponse, error) {
	// Map domain.TransferRequest to Korapay SingleDisbursementRequest
	// Extract customer email if available (not in TransferRequest, defaulting to empty)
	customerEmail := ""

	// Create bank account destination
	bankAccount := &BankAccountDestination{
		Bank:    req.ToBank,
		Account: req.ToAccount,
	}

	// Create destination
	destination := DisbursementDestination{
		Type:        "bank_account",
		Amount:      req.Amount,
		Currency:    "NGN", // Default to NGN, can be enhanced to extract from account details
		Narration:   req.Remark,
		BankAccount: bankAccount,
		Customer: Customer{
			Name:  req.ToClient,
			Email: customerEmail,
		},
	}

	// Create request
	koraRequest := SingleDisbursementRequest{
		Reference:   req.Reference,
		Destination: destination,
	}

	// Call Korapay API
	koraResponse, err := p.client.SendSingleDisbursement(koraRequest)
	if err != nil {
		return nil, fmt.Errorf("korapay single disbursement failed: %w", err)
	}

	// Map Korapay response to domain.TransferResponse
	response := &domain.TransferResponse{
		Status:  koraResponse.Status,
		Message: koraResponse.Message,
	}

	// Map response data
	if koraResponse.Data.Reference != "" {
		response.Data = &domain.TransferData{
			Reference: koraResponse.Data.Reference,
			TxnId:     koraResponse.Data.Reference, // Korapay uses reference as transaction ID
		}
	}

	return response, nil
}

