package paystack

import (
	"context"
	"fmt"
	"strconv"

	"payflow/internal/domain"
	"payflow/internal/service/provider"
)

// Compile-time interface checks
var (
	_ provider.TransferProvider = (*paystackTransferProvider)(nil)
	_ provider.BulkTransferrer  = (*paystackTransferProvider)(nil)
)

// paystackTransferProvider implements TransferProvider and BulkTransferrer for Paystack.
type paystackTransferProvider struct {
	client *Client
}

// NewTransferProvider creates a new Paystack transfer provider.
func NewTransferProvider(client *Client) *paystackTransferProvider {
	return &paystackTransferProvider{client: client}
}

// Name returns the provider identifier.
func (p *paystackTransferProvider) Name() domain.ProviderName {
	return domain.ProviderPaystack
}

// InitiateTransfer creates a recipient on-the-fly and initiates a transfer.
func (p *paystackTransferProvider) InitiateTransfer(ctx context.Context, req *domain.SingleTransferRequest) (*domain.TransferResult, error) {
	currency := req.Currency
	if currency == "" {
		currency = "NGN"
	}

	// Step 1: Create transfer recipient
	recipientResp, err := p.client.CreateTransferRecipient(ctx, CreateRecipientRequest{
		Type:          "nuban",
		Name:          req.AccountName,
		AccountNumber: req.AccountNumber,
		BankCode:      req.BankCode,
		Currency:      currency,
	})
	if err != nil {
		return nil, fmt.Errorf("paystack create recipient failed: %w", err)
	}
	if !recipientResp.Status || recipientResp.Data == nil {
		return &domain.TransferResult{
			Success:   false,
			Reference: req.Reference,
			Status:    "failed",
			Message:   recipientResp.Message,
			Provider:  domain.ProviderPaystack,
			Currency:  currency,
		}, nil
	}

	// Step 2: Parse amount (string in kobo to int64)
	amount, err := strconv.ParseInt(req.Amount, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount '%s': %w", req.Amount, err)
	}

	// Step 3: Initiate transfer
	transferResp, err := p.client.InitiateTransfer(ctx, TransferRequest{
		Source:    "balance",
		Amount:    amount,
		Recipient: recipientResp.Data.RecipientCode,
		Reference: req.Reference,
		Reason:    req.Narration,
		Currency:  currency,
	})
	if err != nil {
		return nil, fmt.Errorf("paystack transfer failed: %w", err)
	}

	return p.mapSingleResponse(req.Reference, currency, transferResp), nil
}

func (p *paystackTransferProvider) mapSingleResponse(reference, currency string, resp *TransferResponse) *domain.TransferResult {
	result := &domain.TransferResult{
		Reference: reference,
		Provider:  domain.ProviderPaystack,
		Currency:  currency,
	}

	if resp.Status {
		result.Success = true
		result.Status = "processing"
		result.Message = resp.Message
		if resp.Data != nil {
			result.TransactionID = resp.Data.TransferCode
			if resp.Data.Status != "" {
				result.Status = resp.Data.Status
			}
		}
	} else {
		result.Success = false
		result.Status = "failed"
		result.Message = resp.Message
	}

	return result
}

// MinBatchSize returns the minimum batch size for Paystack bulk transfers.
func (p *paystackTransferProvider) MinBatchSize() int {
	return 2
}

// MaxBatchSize returns the maximum batch size (Paystack allows 100 per batch).
func (p *paystackTransferProvider) MaxBatchSize() int {
	return 100
}

// InitiateBulkTransfer initiates a bulk transfer via Paystack.
// Each transfer in the batch needs a recipient_code, so we create recipients first.
func (p *paystackTransferProvider) InitiateBulkTransfer(ctx context.Context, req *domain.BulkTransferRequest) (*domain.BulkTransferResult, error) {
	currency := req.Currency
	if currency == "" {
		currency = "NGN"
	}

	// Build bulk items: create recipients and map to Paystack format
	items := make([]BulkTransferItem, 0, len(req.Transfers))
	var results []domain.TransferResult

	for _, t := range req.Transfers {
		// Create recipient for each transfer
		recipientResp, err := p.client.CreateTransferRecipient(ctx, CreateRecipientRequest{
			Type:          "nuban",
			Name:          t.AccountName,
			AccountNumber: t.AccountNumber,
			BankCode:      t.BankCode,
			Currency:      currency,
		})
		if err != nil || !recipientResp.Status || recipientResp.Data == nil {
			msg := "failed to create recipient"
			if err != nil {
				msg = err.Error()
			} else if recipientResp != nil {
				msg = recipientResp.Message
			}
			results = append(results, domain.TransferResult{
				Reference: t.Reference,
				Provider:  domain.ProviderPaystack,
				Success:   false,
				Status:    "failed",
				Message:   msg,
			})
			continue
		}

		amount, err := strconv.ParseInt(t.Amount, 10, 64)
		if err != nil {
			results = append(results, domain.TransferResult{
				Reference: t.Reference,
				Provider:  domain.ProviderPaystack,
				Success:   false,
				Status:    "failed",
				Message:   fmt.Sprintf("invalid amount: %s", t.Amount),
			})
			continue
		}

		items = append(items, BulkTransferItem{
			Amount:    amount,
			Recipient: recipientResp.Data.RecipientCode,
			Reference: t.Reference,
			Reason:    t.Narration,
		})
	}

	// If no valid items, return early with failures
	if len(items) == 0 {
		return &domain.BulkTransferResult{
			Success:        false,
			BatchReference: req.BatchReference,
			Provider:       domain.ProviderPaystack,
			Status:         "failed",
			Message:        "all transfers failed recipient creation",
			TransferResults: results,
		}, nil
	}

	// Execute bulk transfer
	bulkResp, err := p.client.InitiateBulkTransfer(ctx, BulkTransferRequest{
		Source:    "balance",
		Currency:  currency,
		Transfers: items,
	})
	if err != nil {
		return nil, fmt.Errorf("paystack bulk transfer failed: %w", err)
	}

	bulkResult := &domain.BulkTransferResult{
		BatchReference:  req.BatchReference,
		Provider:        domain.ProviderPaystack,
		Currency:        currency,
		TransferResults: results,
	}

	if bulkResp.Status {
		bulkResult.Success = true
		bulkResult.Status = "processing"
		bulkResult.Message = bulkResp.Message
	} else {
		bulkResult.Success = false
		bulkResult.Status = "failed"
		bulkResult.Message = bulkResp.Message
	}

	return bulkResult, nil
}
