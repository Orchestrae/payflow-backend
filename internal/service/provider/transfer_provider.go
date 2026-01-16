package provider

import (
	"context"
	"payflow/internal/domain"
)

// TransferProvider defines the core interface for all transfer providers.
// All provider implementations (Korapay, VFD, etc.) must implement this interface.
// The interface is intentionally minimal - providers implement what they support.
type TransferProvider interface {
	// Name returns the identifier of the provider (e.g., "korapay", "vfd")
	Name() domain.ProviderName

	// InitiateTransfer initiates a transfer using the unified request format.
	// Each provider is responsible for mapping the unified request to its specific API format.
	InitiateTransfer(ctx context.Context, req *domain.SingleTransferRequest) (*domain.TransferResult, error)
}

// BulkTransferrer is an optional interface for providers that support native bulk transfers.
// Providers that implement this can process multiple transfers in a single API call.
// Providers that don't implement this will fall back to concurrent single transfers.
type BulkTransferrer interface {
	// InitiateBulkTransfer processes multiple transfers in a single batch.
	// Returns results for the entire batch.
	InitiateBulkTransfer(ctx context.Context, req *domain.BulkTransferRequest) (*domain.BulkTransferResult, error)

	// MaxBatchSize returns the maximum number of transfers allowed in a single batch.
	// Korapay allows 2-50 per batch.
	MaxBatchSize() int

	// MinBatchSize returns the minimum number of transfers required for a batch.
	// Korapay requires at least 2.
	MinBatchSize() int
}

// AccountEnquirer is an optional interface for providers that support account enquiry.
// Not all providers support this (e.g., Korapay doesn't).
type AccountEnquirer interface {
	// AccountEnquiry gets account details for a given account number
	AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error)
}

// BeneficiaryEnquirer is an optional interface for providers that support beneficiary enquiry.
// Not all providers support this (e.g., Korapay doesn't).
type BeneficiaryEnquirer interface {
	// BeneficiaryEnquiry gets beneficiary details for a transfer
	BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error)
}

// BankLister is an optional interface for providers that support bank listing.
// Not all providers support this (e.g., Korapay doesn't).
type BankLister interface {
	// GetBankList gets the list of all Nigerian banks
	GetBankList(ctx context.Context) (*domain.BankListResponse, error)
}
