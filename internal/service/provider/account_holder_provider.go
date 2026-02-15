package provider

import (
	"context"
	"payflow/internal/domain"
)

// AccountHolderProvider defines the interface for account holder/KYC operations.
// All provider implementations (Korapay, VFD, etc.) must implement this interface.
type AccountHolderProvider interface {
	// Name returns the identifier of the provider (e.g., "korapay", "vfd")
	Name() domain.ProviderName

	// CreateAccountHolder creates an account holder for KYC onboarding.
	// Each provider is responsible for mapping the unified request to its specific API format.
	CreateAccountHolder(ctx context.Context, req *domain.CreateAccountHolderRequest) (*domain.AccountHolderResult, error)

	// GetAccountHolderDetails retrieves account holder details by reference.
	GetAccountHolderDetails(ctx context.Context, reference string) (*domain.AccountHolderDetails, error)

	// UpdateAccountHolderKYC updates account holder KYC information.
	UpdateAccountHolderKYC(ctx context.Context, reference string, req *domain.UpdateAccountHolderKYCRequest) (*domain.UpdateAccountHolderKYCResult, error)

	// GenerateFileUploadURL generates a pre-signed S3 URL for file uploads (KYC documents).
	GenerateFileUploadURL(ctx context.Context, req *domain.GenerateFileUploadURLRequest) (*domain.FileUploadURLResult, error)
}
