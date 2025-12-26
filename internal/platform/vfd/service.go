// internal/platform/vfd/service.go
package vfd

import (
	"context"
	"fmt"
	"net/http"
	"payflow/internal/domain"
)

// VFDService defines the high-level contract for interacting with VFD bank services.
// This is the interface the rest of the PayFlow application will depend on.
type VFDService interface {
	// CreateNewCorporateAccount creates a brand new corporate bank account.
	CreateNewCorporateAccount(ctx context.Context, details NewAccountDetails) (*CorporateAccount, error)
	// CreateDuplicateCorporateAccount creates a new account for an existing VFD client.
	CreateDuplicateCorporateAccount(ctx context.Context, previousAccountNumber string) (*CorporateAccount, error)
	// RetriggerWebhookNotification retriggers a webhook notification via VFD API
	RetriggerWebhookNotification(ctx context.Context, req *domain.VFDRetriggerRequest) (*domain.VFDRetriggerResponse, error)
	// AccountEnquiry gets account details for a given account number
	AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error)
	// BeneficiaryEnquiry gets beneficiary details for a transfer
	BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error)
	// GetBankList gets the list of all Nigerian banks
	GetBankList(ctx context.Context) (*domain.BankListResponse, error)
	// InitiateTransfer initiates a transfer
	InitiateTransfer(ctx context.Context, req *domain.TransferRequest) (*domain.TransferResponse, error)
}

// vfdService is the concrete implementation of the VFDService interface.
type vfdService struct {
	client *Client
}

// NewVFDService creates a new VFD service instance.
func NewVFDService(client *Client) VFDService {
	return &vfdService{client: client}
}

// CreateNewCorporateAccount implements the VFDService interface.
func (s *vfdService) CreateNewCorporateAccount(ctx context.Context, details NewAccountDetails) (*CorporateAccount, error) {
	fmt.Printf("=== VFD Corporate Account Creation ===\n")
	fmt.Printf("RC Number: %s\n", details.RCNumber)
	fmt.Printf("Company Name: %s\n", details.CompanyName)
	fmt.Printf("Incorporation Date: %s\n", details.IncorporationDate.Format("02 January 2006"))
	fmt.Printf("Director BVN: %s\n", details.DirectorBVN)
	fmt.Printf("=====================================\n")

	req := CreateCorporateRequest{
		RCNumber:          details.RCNumber,
		CompanyName:       details.CompanyName,
		BVN:               details.DirectorBVN,
		IncorporationDate: details.IncorporationDate.Format("02 January 2006"), // Format date as required by VFD.
	}
	return s.createAccount(ctx, req)
}

// CreateDuplicateCorporateAccount implements the VFDService interface.
func (s *vfdService) CreateDuplicateCorporateAccount(ctx context.Context, previousAccountNumber string) (*CorporateAccount, error) {
	req := CreateCorporateRequest{
		PreviousAccountNo: previousAccountNumber,
	}
	return s.createAccount(ctx, req)
}

// createAccount is an internal helper that calls the client and handles the response.
func (s *vfdService) createAccount(ctx context.Context, req CreateCorporateRequest) (*CorporateAccount, error) {
	token, err := s.client.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get vfd access token: %w", err)
	}

	headers := map[string]string{
		"AccessToken": token,
	}

	var accountData corporateAccountData
	err = s.client.do(ctx, http.MethodPost, "/corporateclient/create", headers, req, &accountData)
	if err != nil {
		// The `do` method has already translated VFD errors, so we just pass them up.
		return nil, err
	}

	return &CorporateAccount{
		AccountNumber: accountData.AccountNo,
		AccountName:   accountData.AccountName,
	}, nil
}

// RetriggerWebhookNotification implements the VFDService interface.
func (s *vfdService) RetriggerWebhookNotification(ctx context.Context, req *domain.VFDRetriggerRequest) (*domain.VFDRetriggerResponse, error) {
	token, err := s.client.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get vfd access token: %w", err)
	}

	headers := map[string]string{
		"AccessToken": token,
	}

	var response domain.VFDRetriggerResponse
	err = s.client.do(ctx, http.MethodPost, "/transactions/repush", headers, req, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// AccountEnquiry implements the VFDService interface.
func (s *vfdService) AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	token, err := s.client.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get vfd access token: %w", err)
	}

	headers := map[string]string{
		"AccessToken": token,
	}

	var response domain.AccountEnquiryResponse
	path := "/account/enquiry"
	if accountNumber != "" {
		path += "?accountNumber=" + accountNumber
	}

	err = s.client.do(ctx, http.MethodGet, path, headers, nil, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// BeneficiaryEnquiry implements the VFDService interface.
func (s *vfdService) BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	token, err := s.client.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get vfd access token: %w", err)
	}

	headers := map[string]string{
		"AccessToken": token,
	}

	var response domain.BeneficiaryEnquiryResponse
	path := fmt.Sprintf("/transfer/recipient?accountNo=%s&bank=%s&transfer_type=%s", accountNo, bank, transferType)

	err = s.client.do(ctx, http.MethodGet, path, headers, nil, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetBankList implements the VFDService interface.
func (s *vfdService) GetBankList(ctx context.Context) (*domain.BankListResponse, error) {
	token, err := s.client.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get vfd access token: %w", err)
	}

	headers := map[string]string{
		"AccessToken": token,
	}

	var response domain.BankListResponse
	err = s.client.do(ctx, http.MethodGet, "/bank", headers, nil, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// InitiateTransfer implements the VFDService interface.
func (s *vfdService) InitiateTransfer(ctx context.Context, req *domain.TransferRequest) (*domain.TransferResponse, error) {
	token, err := s.client.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get vfd access token: %w", err)
	}

	headers := map[string]string{
		"AccessToken": token,
	}

	var response domain.TransferResponse
	err = s.client.do(ctx, http.MethodPost, "/transfer", headers, req, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
