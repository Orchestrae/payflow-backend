package provider

import (
	"context"
	"payflow/internal/domain"
)

// TransferProvider defines the unified interface for all transfer providers.
// All provider implementations (VFD, Korapay, etc.) must implement this interface.
type TransferProvider interface {
	// Name returns the identifier of the provider (e.g., "vfd", "korapay")
	Name() string

	// AccountEnquiry gets account details for a given account number
	AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error)

	// BeneficiaryEnquiry gets beneficiary details for a transfer
	BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error)

	// GetBankList gets the list of all Nigerian banks
	GetBankList(ctx context.Context) (*domain.BankListResponse, error)

	// InitiateTransfer initiates a transfer
	InitiateTransfer(ctx context.Context, businessID uint, req *domain.TransferRequest) (*domain.TransferResponse, error)
}

