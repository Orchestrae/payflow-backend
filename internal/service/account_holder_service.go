package service

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/service/provider"
)

// AccountHolderService defines the business logic for account holder/KYC operations.
type AccountHolderService interface {
	// CreateAccountHolder creates an account holder for KYC onboarding
	CreateAccountHolder(ctx context.Context, req *domain.CreateAccountHolderRequest) (*domain.AccountHolderResult, error)

	// GetAccountHolderDetails retrieves account holder details by reference
	GetAccountHolderDetails(ctx context.Context, reference string) (*domain.AccountHolderDetails, error)

	// UpdateAccountHolderKYC updates account holder KYC information
	UpdateAccountHolderKYC(ctx context.Context, reference string, req *domain.UpdateAccountHolderKYCRequest) (*domain.UpdateAccountHolderKYCResult, error)

	// GenerateFileUploadURL generates a pre-signed S3 URL for file uploads (KYC documents)
	GenerateFileUploadURL(ctx context.Context, req *domain.GenerateFileUploadURLRequest) (*domain.FileUploadURLResult, error)
}

// accountHolderService implements AccountHolderService
type accountHolderService struct {
	accountHolderProvider provider.AccountHolderProvider
}

// NewAccountHolderService creates a new account holder service
func NewAccountHolderService(accountHolderProvider provider.AccountHolderProvider) AccountHolderService {
	return &accountHolderService{
		accountHolderProvider: accountHolderProvider,
	}
}

// CreateAccountHolder creates an account holder for KYC onboarding
func (s *accountHolderService) CreateAccountHolder(ctx context.Context, req *domain.CreateAccountHolderRequest) (*domain.AccountHolderResult, error) {
	// Validate required fields (basic validation - detailed validation should be in handler)
	if req.FirstName == "" || req.LastName == "" || req.Email == "" {
		return nil, domain.ErrValidationFailed
	}

	// Delegate to provider
	result, err := s.accountHolderProvider.CreateAccountHolder(ctx, req)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetAccountHolderDetails retrieves account holder details by reference
func (s *accountHolderService) GetAccountHolderDetails(ctx context.Context, reference string) (*domain.AccountHolderDetails, error) {
	if reference == "" {
		return nil, domain.ErrValidationFailed
	}

	// Delegate to provider
	details, err := s.accountHolderProvider.GetAccountHolderDetails(ctx, reference)
	if err != nil {
		return nil, err
	}

	return details, nil
}

// UpdateAccountHolderKYC updates account holder KYC information
func (s *accountHolderService) UpdateAccountHolderKYC(ctx context.Context, reference string, req *domain.UpdateAccountHolderKYCRequest) (*domain.UpdateAccountHolderKYCResult, error) {
	if reference == "" {
		return nil, domain.ErrValidationFailed
	}

	// Validate required fields
	if req.FirstName == "" || req.LastName == "" {
		return nil, domain.ErrValidationFailed
	}

	// Delegate to provider
	result, err := s.accountHolderProvider.UpdateAccountHolderKYC(ctx, reference, req)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GenerateFileUploadURL generates a pre-signed S3 URL for file uploads (KYC documents)
func (s *accountHolderService) GenerateFileUploadURL(ctx context.Context, req *domain.GenerateFileUploadURLRequest) (*domain.FileUploadURLResult, error) {
	// Validate required fields
	if req.Reference == "" || req.Purpose == "" || req.ContentType == "" {
		return nil, domain.ErrValidationFailed
	}

	// Delegate to provider
	result, err := s.accountHolderProvider.GenerateFileUploadURL(ctx, req)
	if err != nil {
		return nil, err
	}

	return result, nil
}
