// internal/platform/vfd/service.go
package vfd

import (
	"context"
	"crypto/sha512"
	"fmt"
	"log/slog"
	"net/http"

	"payflow/internal/domain"
)

// VFDService defines the high-level contract for interacting with VFD bank services.
type VFDService interface {
	// Account Operations
	CreateNewCorporateAccount(ctx context.Context, details NewAccountDetails) (*CorporateAccount, error)
	CreateDuplicateCorporateAccount(ctx context.Context, previousAccountNumber string) (*CorporateAccount, error)

	// Transfer Operations (implements the standard provider pattern)
	AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error)
	BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error)
	GetBankList(ctx context.Context) (*domain.BankListResponse, error)
	InitiateTransfer(ctx context.Context, req *domain.TransferRequest) (*domain.TransferResponse, error)
	GetTransactionStatus(ctx context.Context, reference, sessionID string) (*domain.TransactionStatusResponse, error)

	// Webhook Operations
	RetriggerWebhookNotification(ctx context.Context, req *domain.VFDRetriggerRequest) (*domain.VFDRetriggerResponse, error)
}

// vfdService is the concrete implementation of the VFDService interface.
type vfdService struct {
	client *Client
}

// NewVFDService creates a new VFD service instance.
func NewVFDService(client *Client) VFDService {
	return &vfdService{client: client}
}

// ============================================================================
// Corporate Account Operations
// ============================================================================

// CreateNewCorporateAccount creates a new corporate bank account.
func (s *vfdService) CreateNewCorporateAccount(ctx context.Context, details NewAccountDetails) (*CorporateAccount, error) {
	slog.Info("Creating VFD corporate account",
		"rc_number", details.RCNumber,
		"company_name", details.CompanyName,
	)

	req := CreateCorporateRequest{
		RCNumber:          details.RCNumber,
		CompanyName:       details.CompanyName,
		BVN:               details.DirectorBVN,
		IncorporationDate: details.IncorporationDate.Format("02 January 2006"),
	}

	var accountData CorporateAccountData
	err := s.client.doAuthenticatedRequest(ctx, http.MethodPost, "/corporateclient/create", req, &accountData)
	if err != nil {
		return nil, err
	}

	return &CorporateAccount{
		AccountNumber: accountData.AccountNo,
		AccountName:   accountData.AccountName,
	}, nil
}

// CreateDuplicateCorporateAccount creates a new account for an existing VFD client.
func (s *vfdService) CreateDuplicateCorporateAccount(ctx context.Context, previousAccountNumber string) (*CorporateAccount, error) {
	req := CreateCorporateRequest{
		PreviousAccountNo: previousAccountNumber,
	}

	var accountData CorporateAccountData
	err := s.client.doAuthenticatedRequest(ctx, http.MethodPost, "/corporateclient/create", req, &accountData)
	if err != nil {
		return nil, err
	}

	return &CorporateAccount{
		AccountNumber: accountData.AccountNo,
		AccountName:   accountData.AccountName,
	}, nil
}

// ============================================================================
// Transfer Operations
// ============================================================================

// AccountEnquiry gets account details for a given account number.
// If accountNumber is empty, returns the pool account details.
func (s *vfdService) AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	path := "/account/enquiry"
	if accountNumber != "" {
		path += "?accountNumber=" + accountNumber
	}

	var data domain.AccountEnquiryData
	err := s.client.doAuthenticatedRequest(ctx, http.MethodGet, path, nil, &data)
	if err != nil {
		return nil, fmt.Errorf("account enquiry failed: %w", err)
	}

	return &domain.AccountEnquiryResponse{
		Status:  "00",
		Message: "Account Details",
		Data:    &data,
	}, nil
}

// BeneficiaryEnquiry gets the transfer recipient's account details.
func (s *vfdService) BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	path := fmt.Sprintf("/transfer/recipient?accountNo=%s&bank=%s&transfer_type=%s", accountNo, bank, transferType)

	var data domain.BeneficiaryEnquiryData
	err := s.client.doAuthenticatedRequest(ctx, http.MethodGet, path, nil, &data)
	if err != nil {
		return nil, fmt.Errorf("beneficiary enquiry failed: %w", err)
	}

	return &domain.BeneficiaryEnquiryResponse{
		Status:  "00",
		Message: "Account Found",
		Data:    &data,
	}, nil
}

// GetBankList gets the list of all Nigerian banks and bank codes.
func (s *vfdService) GetBankList(ctx context.Context) (*domain.BankListResponse, error) {
	var data []domain.BankData
	err := s.client.doAuthenticatedRequest(ctx, http.MethodGet, "/bank", nil, &data)
	if err != nil {
		return nil, fmt.Errorf("get bank list failed: %w", err)
	}

	return &domain.BankListResponse{
		Status:  "00",
		Message: "Bank List",
		Data:    data,
	}, nil
}

// InitiateTransfer initiates a funds transfer.
// Note: The caller should have already:
// 1. Called AccountEnquiry to get sender details
// 2. Called BeneficiaryEnquiry to get recipient details
// 3. Generated the signature
func (s *vfdService) InitiateTransfer(ctx context.Context, req *domain.TransferRequest) (*domain.TransferResponse, error) {
	slog.Info("Initiating VFD transfer",
		"from_account", req.FromAccount,
		"to_account", req.ToAccount,
		"amount", req.Amount,
		"reference", req.Reference,
		"transfer_type", req.TransferType,
	)

	// Build VFD transfer request
	vfdReq := TransferRequest{
		FromAccount:           req.FromAccount,
		UniqueSenderAccountId: req.UniqueSenderAccountId,
		FromClientId:          req.FromClientId,
		FromClient:            req.FromClient,
		FromSavingsId:         req.FromSavingsId,
		FromBvn:               req.FromBvn,
		ToClientId:            req.ToClientId,
		ToClient:              req.ToClient,
		ToSavingsId:           req.ToSavingsId,
		ToSession:             req.ToSession,
		ToBvn:                 req.ToBvn,
		ToAccount:             req.ToAccount,
		ToBank:                req.ToBank,
		Signature:             req.Signature,
		Amount:                req.Amount,
		Remark:                req.Remark,
		TransferType:          req.TransferType,
		Reference:             req.Reference,
	}

	var data domain.TransferData
	err := s.client.doAuthenticatedRequest(ctx, http.MethodPost, "/transfer", vfdReq, &data)
	if err != nil {
		return nil, fmt.Errorf("VFD transfer failed: %w", err)
	}

	return &domain.TransferResponse{
		Status:  "00",
		Message: "Successful Transfer",
		Data:    &data,
	}, nil
}

// GetTransactionStatus queries the status of a transaction.
// Provide either reference OR sessionId (not both).
func (s *vfdService) GetTransactionStatus(ctx context.Context, reference, sessionID string) (*domain.TransactionStatusResponse, error) {
	var path string
	if reference != "" {
		path = "/transactions?reference=" + reference
	} else if sessionID != "" {
		path = "/transactions?sessionId=" + sessionID
	} else {
		return nil, fmt.Errorf("either reference or sessionId is required")
	}

	var data domain.TransactionStatusData
	err := s.client.doAuthenticatedRequest(ctx, http.MethodGet, path, nil, &data)
	if err != nil {
		return nil, fmt.Errorf("transaction status query failed: %w", err)
	}

	return &domain.TransactionStatusResponse{
		Status:  "00",
		Message: "Successful Transaction Retrieval",
		Data:    &data,
	}, nil
}

// ============================================================================
// Webhook Operations
// ============================================================================

// RetriggerWebhookNotification retriggers a webhook notification.
func (s *vfdService) RetriggerWebhookNotification(ctx context.Context, req *domain.VFDRetriggerRequest) (*domain.VFDRetriggerResponse, error) {
	var response domain.VFDRetriggerResponse
	err := s.client.doAuthenticatedRequest(ctx, http.MethodPost, "/transactions/repush", req, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// ============================================================================
// Utility Functions
// ============================================================================

// GenerateTransferSignature generates the SHA512 signature for VFD transfers.
// signature = SHA512(fromAccount + toAccount)
func GenerateTransferSignature(fromAccount, toAccount string) string {
	data := fromAccount + toAccount
	hash := sha512.Sum512([]byte(data))
	return fmt.Sprintf("%x", hash)
}
