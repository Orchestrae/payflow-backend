package provider

import (
	"context"
	"fmt"
	"log/slog"

	"payflow/internal/domain"
	"payflow/internal/platform/vfd"
)

// Ensure VFDProvider implements all required interfaces
var (
	_ TransferProvider    = (*VFDProvider)(nil)
	_ AccountEnquirer     = (*VFDProvider)(nil)
	_ BeneficiaryEnquirer = (*VFDProvider)(nil)
	_ BankLister          = (*VFDProvider)(nil)
)

// VFDProvider is an adapter that wraps the VFD service to implement the TransferProvider interface.
type VFDProvider struct {
	vfdService vfd.VFDService
}

// NewVFDProvider creates a new VFD provider adapter.
func NewVFDProvider(vfdService vfd.VFDService) *VFDProvider {
	return &VFDProvider{
		vfdService: vfdService,
	}
}

// Name returns the provider identifier.
func (p *VFDProvider) Name() domain.ProviderName {
	return domain.ProviderVFD
}

// ============================================================================
// TransferProvider Implementation
// ============================================================================

// InitiateTransfer implements the TransferProvider interface.
// VFD requires a multi-step flow:
// 1. AccountEnquiry - get sender (pool) account details
// 2. BeneficiaryEnquiry - get recipient details
// 3. Generate signature - SHA512(fromAccount + toAccount)
// 4. InitiateTransfer - execute the transfer
func (p *VFDProvider) InitiateTransfer(ctx context.Context, req *domain.SingleTransferRequest) (*domain.TransferResult, error) {
	slog.Info("VFD: Starting transfer flow",
		"reference", req.Reference,
		"to_account", req.AccountNumber,
		"bank_code", req.BankCode,
		"amount", req.Amount,
	)

	// Step 1: Get sender (pool) account details
	senderDetails, err := p.getSenderDetails(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender details: %w", err)
	}

	// Step 2: Get recipient details
	transferType := p.determineTransferType(req.BankCode)
	recipientDetails, err := p.getRecipientDetails(ctx, req.AccountNumber, req.BankCode, transferType)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipient details: %w", err)
	}

	// Step 3: Generate signature
	signature := vfd.GenerateTransferSignature(senderDetails.AccountNo, recipientDetails.Account.Number)

	// Step 4: Build and execute transfer
	vfdReq := p.buildTransferRequest(req, senderDetails, recipientDetails, signature, transferType)

	vfdResp, err := p.vfdService.InitiateTransfer(ctx, vfdReq)
	if err != nil {
		return nil, fmt.Errorf("VFD transfer failed: %w", err)
	}

	// Step 5: Map response to unified format
	return p.mapTransferResponse(req.Reference, vfdResp), nil
}

// getSenderDetails retrieves the pool account details (sender).
func (p *VFDProvider) getSenderDetails(ctx context.Context) (*domain.AccountEnquiryData, error) {
	// Empty account number returns the pool account
	resp, err := p.vfdService.AccountEnquiry(ctx, "")
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, fmt.Errorf("no sender account data returned")
	}

	slog.Debug("VFD: Got sender details",
		"account_no", resp.Data.AccountNo,
		"client", resp.Data.Client,
	)

	return resp.Data, nil
}

// getRecipientDetails retrieves the recipient account details.
func (p *VFDProvider) getRecipientDetails(ctx context.Context, accountNo, bankCode, transferType string) (*domain.BeneficiaryEnquiryData, error) {
	resp, err := p.vfdService.BeneficiaryEnquiry(ctx, accountNo, bankCode, transferType)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, fmt.Errorf("no recipient data returned for account %s", accountNo)
	}

	slog.Debug("VFD: Got recipient details",
		"name", resp.Data.Name,
		"account", resp.Data.Account.Number,
		"bank", resp.Data.Bank,
	)

	return resp.Data, nil
}

// determineTransferType determines if transfer is intra (VFD-VFD) or inter (VFD-Other).
// Bank code 999999 is VFD Microfinance Bank.
func (p *VFDProvider) determineTransferType(bankCode string) string {
	if bankCode == "999999" {
		return "intra"
	}
	return "inter"
}

// buildTransferRequest builds the VFD transfer request from all gathered details.
func (p *VFDProvider) buildTransferRequest(
	req *domain.SingleTransferRequest,
	sender *domain.AccountEnquiryData,
	recipient *domain.BeneficiaryEnquiryData,
	signature string,
	transferType string,
) *domain.TransferRequest {
	vfdReq := &domain.TransferRequest{
		// Sender (From) details - from AccountEnquiry
		FromAccount:   sender.AccountNo,
		FromClientId:  sender.ClientId,
		FromClient:    sender.Client,
		FromSavingsId: sender.AccountId,

		// Recipient (To) details - from BeneficiaryEnquiry
		ToAccount:  recipient.Account.Number,
		ToBank:     req.BankCode,
		ToClient:   recipient.Name,
		ToClientId: recipient.ClientId,
		ToBvn:      recipient.BVN,

		// Transfer details
		Signature:    signature,
		Amount:       req.Amount,
		Remark:       req.Narration,
		TransferType: transferType,
		Reference:    req.Reference,
	}

	// Set intra-specific or inter-specific fields
	if transferType == "intra" {
		vfdReq.ToSavingsId = recipient.Account.ID
	} else {
		// For inter-bank, toSession is the account ID
		vfdReq.ToSession = recipient.Account.ID
	}

	return vfdReq
}

// mapTransferResponse maps VFD response to unified TransferResult.
func (p *VFDProvider) mapTransferResponse(reference string, resp *domain.TransferResponse) *domain.TransferResult {
	result := &domain.TransferResult{
		Reference: reference,
		Provider:  domain.ProviderVFD,
		Currency:  "NGN",
		Status:    resp.Status,
		Message:   resp.Message,
	}

	// VFD uses "00" for success
	if resp.Status == "00" {
		result.Success = true
		result.Status = "success"
		if resp.Data != nil {
			result.TransactionID = resp.Data.TxnId
		}
	} else {
		result.Success = false
		result.Status = "failed"
	}

	return result
}

// ============================================================================
// Optional Capability Interfaces
// ============================================================================

// AccountEnquiry implements the AccountEnquirer interface.
func (p *VFDProvider) AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	return p.vfdService.AccountEnquiry(ctx, accountNumber)
}

// BeneficiaryEnquiry implements the BeneficiaryEnquirer interface.
func (p *VFDProvider) BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	return p.vfdService.BeneficiaryEnquiry(ctx, accountNo, bank, transferType)
}

// GetBankList implements the BankLister interface.
func (p *VFDProvider) GetBankList(ctx context.Context) (*domain.BankListResponse, error) {
	return p.vfdService.GetBankList(ctx)
}
