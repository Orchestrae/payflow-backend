// internal/platform/vfd/mock_service.go
package vfd

import (
	"context"
	"fmt"
	"time"
)

// MockVFDService is a mock implementation of VFDService for testing purposes
type MockVFDService struct{}

// NewMockVFDService creates a new mock VFD service
func NewMockVFDService() VFDService {
	return &MockVFDService{}
}

// CreateNewCorporateAccount implements the VFDService interface for testing
func (s *MockVFDService) CreateNewCorporateAccount(ctx context.Context, details NewAccountDetails) (*CorporateAccount, error) {
	// Simulate a small delay to mimic real API call
	time.Sleep(100 * time.Millisecond)

	// Generate a mock account number based on the company name
	accountNumber := fmt.Sprintf("1234567890")
	accountName := fmt.Sprintf("%s", details.CompanyName)

	return &CorporateAccount{
		AccountNumber: accountNumber,
		AccountName:   accountName,
	}, nil
}

// CreateDuplicateCorporateAccount implements the VFDService interface for testing
func (s *MockVFDService) CreateDuplicateCorporateAccount(ctx context.Context, previousAccountNumber string) (*CorporateAccount, error) {
	// Simulate a small delay to mimic real API call
	time.Sleep(100 * time.Millisecond)

	// Generate a mock account number
	accountNumber := fmt.Sprintf("9876543210")
	accountName := fmt.Sprintf("DUPLICATE_ACCOUNT_%s", previousAccountNumber)

	return &CorporateAccount{
		AccountNumber: accountNumber,
		AccountName:   accountName,
	}, nil
}
