package service

import (
	"context"
	"fmt"
	"regexp"

	"payflow/internal/platform/paystack"
)

var bvnRegex = regexp.MustCompile(`^\d{11}$`)

// AccountVerificationResult contains the verified bank account details.
type AccountVerificationResult struct {
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	BankCode      string `json:"bank_code"`
	Verified      bool   `json:"verified"`
}

// BVNVerificationResult contains the BVN verification details.
type BVNVerificationResult struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	BVN       string `json:"bvn_last4"` // Only last 4 returned for security
	Verified  bool   `json:"verified"`
}

// AccountVerificationService validates bank account details.
type AccountVerificationService interface {
	VerifyBankAccount(ctx context.Context, bankCode, accountNumber string) (*AccountVerificationResult, error)
	VerifyBVN(ctx context.Context, bvn string) (*BVNVerificationResult, error)
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

func (s *accountVerificationService) VerifyBVN(ctx context.Context, bvn string) (*BVNVerificationResult, error) {
	if s.paystackClient == nil {
		return nil, fmt.Errorf("BVN verification unavailable: Paystack not configured")
	}

	if !bvnRegex.MatchString(bvn) {
		return nil, fmt.Errorf("invalid BVN: must be exactly 11 digits")
	}

	resp, err := s.paystackClient.ResolveBVN(ctx, bvn)
	if err != nil {
		return &BVNVerificationResult{
			BVN:      bvn[len(bvn)-4:],
			Verified: false,
		}, nil
	}

	result := &BVNVerificationResult{
		BVN:      bvn[len(bvn)-4:],
		Verified: resp.Status,
	}
	if resp.Data != nil {
		result.FirstName = resp.Data.FirstName
		result.LastName = resp.Data.LastName
	}

	return result, nil
}
