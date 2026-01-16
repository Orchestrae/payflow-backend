// internal/platform/vfd/mock_service.go
package vfd

import (
	"context"
	"fmt"
	"payflow/internal/domain"
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

// AccountEnquiry implements the VFDService interface for testing
func (s *MockVFDService) AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	// Simulate a small delay to mimic real API call
	time.Sleep(100 * time.Millisecond)

	// If no account number, return pool account
	acctNo := accountNumber
	if acctNo == "" {
		acctNo = "1001554791" // Pool account
	}

	return &domain.AccountEnquiryResponse{
		Status:  "00",
		Message: "Account Details",
		Data: &domain.AccountEnquiryData{
			AccountNo:          acctNo,
			AccountBalance:     "1000000.00",
			AccountId:          "155479",
			Client:             "Mock Pool Client",
			ClientId:           "138421",
			SavingsProductName: "Corporate Current Account",
		},
	}, nil
}

// BeneficiaryEnquiry implements the VFDService interface for testing
func (s *MockVFDService) BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	// Simulate a small delay to mimic real API call
	time.Sleep(100 * time.Millisecond)

	return &domain.BeneficiaryEnquiryResponse{
		Status:  "00",
		Message: "Account Found",
		Data: &domain.BeneficiaryEnquiryData{
			Name:     "Mock Beneficiary",
			ClientId: "456",
			BVN:      "12345678901",
			Account: struct {
				Number string `json:"number"`
				ID     string `json:"id"`
			}{
				Number: accountNo,
				ID:     "654321",
			},
			Status:   "active",
			Currency: "NGN",
			Bank:     "Mock Bank",
		},
	}, nil
}

// GetBankList implements the VFDService interface for testing
func (s *MockVFDService) GetBankList(ctx context.Context) (*domain.BankListResponse, error) {
	// Simulate a small delay to mimic real API call
	time.Sleep(100 * time.Millisecond)

	return &domain.BankListResponse{
		Status:  "00",
		Message: "Bank List",
		Data: []domain.BankData{
			{
				Code: "000004",
				Name: "VFD Microfinance Bank",
			},
			{
				Code: "000001",
				Name: "First Bank of Nigeria",
			},
		},
	}, nil
}

// InitiateTransfer implements the VFDService interface for testing
func (s *MockVFDService) InitiateTransfer(ctx context.Context, req *domain.TransferRequest) (*domain.TransferResponse, error) {
	// Simulate a small delay to mimic real API call
	time.Sleep(100 * time.Millisecond)

	return &domain.TransferResponse{
		Status:  "00",
		Message: "Successful Transfer",
		Data: &domain.TransferData{
			TxnId:     req.Reference,
			SessionId: "mock-session-id",
			Reference: "mock-reference",
		},
	}, nil
}

// RetriggerWebhookNotification implements the VFDService interface for testing
func (s *MockVFDService) RetriggerWebhookNotification(ctx context.Context, req *domain.VFDRetriggerRequest) (*domain.VFDRetriggerResponse, error) {
	// Simulate a small delay to mimic real API call
	time.Sleep(100 * time.Millisecond)

	return &domain.VFDRetriggerResponse{
		Status:  "00",
		Message: "success",
	}, nil
}

// GetTransactionStatus implements the VFDService interface for testing
func (s *MockVFDService) GetTransactionStatus(ctx context.Context, reference, sessionID string) (*domain.TransactionStatusResponse, error) {
	// Simulate a small delay to mimic real API call
	time.Sleep(100 * time.Millisecond)

	txnID := reference
	if txnID == "" {
		txnID = sessionID
	}

	return &domain.TransactionStatusResponse{
		Status:  "00",
		Message: "Successful Transaction Retrieval",
		Data: &domain.TransactionStatusData{
			TxnId:             txnID,
			Amount:            "1000.00",
			AccountNo:         "0000000000",
			FromAccountNo:     "1234567890",
			TransactionStatus: "00",
			TransactionDate:   "2026-01-15 12:00:00.0",
			ToBank:            "999999",
			FromBank:          "999999",
			SessionId:         "mock-session-id",
			BankTransactionId: "mock-bank-txn-id",
			TransactionType:   "OUTFLOW",
		},
	}, nil
}
