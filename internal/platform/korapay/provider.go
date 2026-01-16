package korapay

import (
	"context"
	"fmt"
	"strconv"

	"payflow/internal/domain"
	"payflow/internal/service/provider"
)

// Ensure korapayTransferProvider implements required interfaces
var (
	_ provider.TransferProvider = (*korapayTransferProvider)(nil)
	_ provider.BulkTransferrer  = (*korapayTransferProvider)(nil)
)

// korapayTransferProvider implements the TransferProvider and BulkTransferrer interfaces for Korapay.
type korapayTransferProvider struct {
	client *Client
}

// NewTransferProvider creates a new Korapay transfer provider.
func NewTransferProvider(client *Client) *korapayTransferProvider {
	return &korapayTransferProvider{
		client: client,
	}
}

// Name returns the provider identifier.
func (p *korapayTransferProvider) Name() domain.ProviderName {
	return domain.ProviderKorapay
}

// ============================================================================
// Single Transfer (TransferProvider interface)
// ============================================================================

// InitiateTransfer implements the TransferProvider interface.
// Maps the unified SingleTransferRequest to Korapay's disbursement API format.
func (p *korapayTransferProvider) InitiateTransfer(ctx context.Context, req *domain.SingleTransferRequest) (*domain.TransferResult, error) {
	// Set default currency
	currency := req.Currency
	if currency == "" {
		currency = "NGN"
	}

	// Use business email for Korapay customer email (required by Korapay)
	customerEmail := req.BusinessEmail
	if customerEmail == "" {
		// Fallback to a generic email if business email not available
		customerEmail = fmt.Sprintf("transfer-%d@payflow.local", req.BusinessID)
	}

	// Build the Korapay request
	koraRequest := SingleDisbursementRequest{
		Reference: req.Reference,
		Destination: DisbursementDestination{
			Type:      "bank_account",
			Amount:    req.Amount,
			Currency:  currency,
			Narration: req.Narration,
			BankAccount: &BankAccountDestination{
				Bank:    req.BankCode,
				Account: req.AccountNumber,
			},
			Customer: Customer{
				Name:  req.AccountName,
				Email: customerEmail,
			},
		},
	}

	// Call Korapay API
	koraResponse, err := p.client.SendSingleDisbursement(koraRequest)
	if err != nil {
		return nil, fmt.Errorf("korapay disbursement failed: %w", err)
	}

	// Map response to unified format
	return p.mapSingleResponse(req.Reference, currency, koraResponse), nil
}

// mapSingleResponse maps Korapay single disbursement response to unified TransferResult.
func (p *korapayTransferProvider) mapSingleResponse(reference, currency string, resp *SingleDisbursementResponse) *domain.TransferResult {
	result := &domain.TransferResult{
		Reference: reference,
		Provider:  domain.ProviderKorapay,
		Currency:  currency,
	}

	// Korapay returns status: true/false (boolean) in top level
	if resp.Status {
		result.Success = true
		result.Status = "processing" // Korapay transfers start in processing state
		result.Message = resp.Message

		// Extract data from response if available
		if resp.Data != nil {
			if resp.Data.Reference != "" {
				result.TransactionID = resp.Data.Reference
			}
			if resp.Data.Status != "" {
				result.Status = resp.Data.Status
			}
			if resp.Data.Fee != "" {
				result.Fee = resp.Data.Fee
			}
		}
	} else {
		result.Success = false
		result.Status = "failed"
		result.Message = resp.Message
	}

	return result
}

// ============================================================================
// Bulk Transfer (BulkTransferrer interface)
// ============================================================================

// MinBatchSize returns the minimum number of transfers for a batch.
// Korapay requires at least 2 transfers per batch.
func (p *korapayTransferProvider) MinBatchSize() int {
	return 2
}

// MaxBatchSize returns the maximum number of transfers for a batch.
// Korapay allows up to 50 transfers per batch.
func (p *korapayTransferProvider) MaxBatchSize() int {
	return 50
}

// InitiateBulkTransfer implements the BulkTransferrer interface.
// Uses Korapay's native bulk disbursement endpoint for efficiency.
func (p *korapayTransferProvider) InitiateBulkTransfer(ctx context.Context, req *domain.BulkTransferRequest) (*domain.BulkTransferResult, error) {
	// Set default currency
	currency := req.Currency
	if currency == "" {
		currency = "NGN"
	}

	// Build Korapay bulk payout items
	payouts := make([]BulkPayoutItem, len(req.Transfers))
	for i, transfer := range req.Transfers {
		// Parse amount as float for Korapay
		amount, err := strconv.ParseFloat(transfer.Amount, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount for transfer %s: %w", transfer.Reference, err)
		}

		// Use business email or fallback
		customerEmail := req.BusinessEmail
		if customerEmail == "" {
			customerEmail = fmt.Sprintf("transfer-%d@payflow.local", req.BusinessID)
		}

		payouts[i] = BulkPayoutItem{
			Reference: transfer.Reference,
			Amount:    amount,
			Type:      "bank_account",
			Narration: transfer.Narration,
			BankAccount: &BulkBankAccountDestination{
				BankCode:      transfer.BankCode,
				AccountNumber: transfer.AccountNumber,
			},
			Customer: Customer{
				Name:  transfer.AccountName,
				Email: customerEmail,
			},
		}
	}

	// Build Korapay bulk request
	koraRequest := BulkPayoutRequest{
		BatchReference:    req.BatchReference,
		Description:       req.Description,
		MerchantBearsCost: req.MerchantBearsCost,
		Currency:          currency,
		Payouts:           payouts,
	}

	// Call Korapay bulk API
	koraResponse, err := p.client.SendBulkPayout(koraRequest)
	if err != nil {
		return nil, fmt.Errorf("korapay bulk disbursement failed: %w", err)
	}

	// Map response to unified format
	return p.mapBulkResponse(req.BatchReference, currency, koraResponse), nil
}

// mapBulkResponse maps Korapay bulk payout response to unified BulkTransferResult.
func (p *korapayTransferProvider) mapBulkResponse(batchReference, currency string, resp *BulkPayoutResponse) *domain.BulkTransferResult {
	result := &domain.BulkTransferResult{
		BatchReference: batchReference,
		Provider:       domain.ProviderKorapay,
		Currency:       currency,
	}

	if resp.Status {
		result.Success = true
		result.Status = "pending" // Bulk transfers start as pending
		result.Message = resp.Message

		if resp.Data != nil {
			if resp.Data.Reference != "" {
				result.BatchReference = resp.Data.Reference
			}
			if resp.Data.Status != "" {
				result.Status = resp.Data.Status
			}
			if resp.Data.TotalChargeableAmount > 0 {
				result.TotalChargeableAmount = fmt.Sprintf("%.2f", resp.Data.TotalChargeableAmount)
			}
		}
	} else {
		result.Success = false
		result.Status = "failed"
		result.Message = resp.Message
	}

	return result
}
