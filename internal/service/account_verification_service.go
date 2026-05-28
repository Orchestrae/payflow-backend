package service

import (
	"context"
	"fmt"

	"payflow/internal/platform/paystack"
)

// AccountVerificationResult contains the verified bank account details.
type AccountVerificationResult struct {
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	BankCode      string `json:"bank_code"`
	Verified      bool   `json:"verified"`
}

// AccountVerificationService validates bank account details.
type AccountVerificationService interface {
	VerifyBankAccount(ctx context.Context, bankCode, accountNumber string) (*AccountVerificationResult, error)
}

type accountVerificationService struct {
	paystackClient *paystack.Client
}

// NewAccountVerificationService creates a new account verification service.
func NewAccountVerificationService(paystackClient *paystack.Client) AccountVerificationService {
	return &accountVerificationService{paystackClient: paystackClient}
}

func (s *accountVerificationService) VerifyBankAccount(ctx context.Context, bankCode, accountNumber string) (*AccountVerificationResult, error) {
	if s.paystackClient == nil {
		return nil, fmt.Errorf("account verification unavailable: Paystack not configured")
	}

	resp, err := s.paystackClient.ResolveAccountNumber(ctx, accountNumber, bankCode)
	if err != nil {
		return &AccountVerificationResult{
			AccountNumber: accountNumber,
			BankCode:      bankCode,
			Verified:      false,
		}, nil
	}

	result := &AccountVerificationResult{
		AccountNumber: accountNumber,
		BankCode:      bankCode,
		Verified:      resp.Status,
	}
	if resp.Data != nil {
		result.AccountName = resp.Data.AccountName
	}

	return result, nil
}
