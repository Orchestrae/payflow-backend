package korapay

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"payflow/internal/domain"
	"payflow/internal/service/provider"
)

// Ensure korapayVirtualAccountProvider implements required interfaces
var (
	_ provider.VirtualAccountProvider      = (*korapayVirtualAccountProvider)(nil)
	_ provider.VirtualAccountBalancer      = (*korapayVirtualAccountProvider)(nil)
	_ provider.VirtualAccountTransactionLister = (*korapayVirtualAccountProvider)(nil)
)

// korapayVirtualAccountProvider implements the VirtualAccountProvider interface for Korapay.
type korapayVirtualAccountProvider struct {
	client *Client
}

// NewVirtualAccountProvider creates a new Korapay virtual account provider.
func NewVirtualAccountProvider(client *Client) *korapayVirtualAccountProvider {
	return &korapayVirtualAccountProvider{
		client: client,
	}
}

// Name returns the provider identifier.
func (p *korapayVirtualAccountProvider) Name() domain.ProviderName {
	return domain.ProviderKorapay
}

// ============================================================================
// Virtual Account Provider (VirtualAccountProvider interface)
// ============================================================================

// CreateVirtualAccount implements the VirtualAccountProvider interface.
// Maps the unified CreateVirtualAccountRequest to Korapay's virtual account API format.
func (p *korapayVirtualAccountProvider) CreateVirtualAccount(ctx context.Context, req *domain.CreateVirtualAccountRequest) (*domain.VirtualAccountResult, error) {
	// Generate account reference if not provided
	accountReference := req.AccountReference
	if accountReference == "" {
		accountReference = fmt.Sprintf("payflow-va-%d-%d", req.BusinessID, time.Now().Unix())
	}

	// Set default bank code if not provided (Kora uses "000" for sandbox, "035" for Wema in prod)
	bankCode := req.BankCode
	if bankCode == "" {
		bankCode = "000" // Default to sandbox/test bank code
	}

	// Build the Korapay request
	koraRequest := VirtualAccountCreateRequest{
		AccountName:     req.AccountName,
		AccountReference: accountReference,
		Permanent:       req.Permanent,
		BankCode:        bankCode,
		Customer: VirtualAccountCustomer{
			Name:  req.CustomerName,
			Email: req.CustomerEmail,
		},
		KYC: VirtualAccountKYC{
			BVN: req.BVN,
			NIN: req.NIN,
		},
	}

	// If Permanent not set, default to true (Kora requires this)
	if !koraRequest.Permanent {
		koraRequest.Permanent = true
	}

	// Call Korapay API
	koraResponse, err := p.client.CreateVirtualAccount(koraRequest)
	if err != nil {
		return nil, fmt.Errorf("korapay create virtual account failed: %w", err)
	}

	// Map response to unified format
	return p.mapVirtualAccountResponse(accountReference, koraResponse), nil
}

// GetVirtualAccount implements the VirtualAccountProvider interface.
// Retrieves virtual account details by reference.
func (p *korapayVirtualAccountProvider) GetVirtualAccount(ctx context.Context, accountReference string) (*domain.VirtualAccountResult, error) {
	// Call Korapay API
	koraResponse, err := p.client.GetVirtualAccount(accountReference)
	if err != nil {
		return nil, fmt.Errorf("korapay get virtual account failed: %w", err)
	}

	// Map response to unified format
	var accountRef string
	if koraResponse.Data != nil {
		accountRef = koraResponse.Data.AccountReference
	}
	return p.mapVirtualAccountResponse(accountRef, koraResponse), nil
}

// mapVirtualAccountResponse maps Korapay virtual account response to unified VirtualAccountResult.
func (p *korapayVirtualAccountProvider) mapVirtualAccountResponse(accountReference string, resp *VirtualAccountResponse) *domain.VirtualAccountResult {
	result := &domain.VirtualAccountResult{
		Provider:        domain.ProviderKorapay,
		AccountReference: accountReference,
		Currency:        "NGN",
	}

	// Korapay returns status: true/false (boolean) in top level
	if resp.Status && resp.Data != nil {
		result.Success = true
		result.Message = resp.Message
		result.AccountNumber = resp.Data.AccountNumber
		result.AccountName = resp.Data.AccountName
		result.BankCode = resp.Data.BankCode
		result.BankName = resp.Data.BankName
		result.UniqueID = resp.Data.UniqueID
		result.AccountStatus = resp.Data.AccountStatus
		result.Currency = resp.Data.Currency

		// Parse created_at timestamp
		if resp.Data.CreatedAt != "" {
			if parsedTime, err := time.Parse(time.RFC3339, resp.Data.CreatedAt); err == nil {
				result.CreatedAt = parsedTime
			} else {
				// Fallback to current time if parsing fails
				result.CreatedAt = time.Now()
			}
		} else {
			result.CreatedAt = time.Now()
		}
	} else {
		result.Success = false
		result.Message = resp.Message
		result.CreatedAt = time.Now()
	}

	return result
}

// ============================================================================
// Balance Checking (VirtualAccountBalancer interface)
// ============================================================================

// GetBalance implements the VirtualAccountBalancer interface.
// Kora doesn't have a direct balance endpoint, so we calculate from transactions.
func (p *korapayVirtualAccountProvider) GetBalance(ctx context.Context, accountReference string) (*domain.VirtualAccountBalanceResult, error) {
	// First, get virtual account details to get account number
	accountResp, err := p.GetVirtualAccount(ctx, accountReference)
	if err != nil {
		return nil, fmt.Errorf("failed to get account details for balance: %w", err)
	}

	if accountResp.AccountNumber == "" {
		return nil, fmt.Errorf("account number not found for reference: %s", accountReference)
	}

	// Get transactions to calculate total amount received
	// Note: We get all transactions to calculate total balance
	// In production, you might want to cache this or use a webhook to track balance
	transactionsResp, err := p.ListTransactions(ctx, accountResp.AccountNumber, nil, nil, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for balance calculation: %w", err)
	}

	result := &domain.VirtualAccountBalanceResult{
		Success:         true,
		Provider:        domain.ProviderKorapay,
		AccountNumber:   accountResp.AccountNumber,
		AccountReference: accountReference,
		Currency:        "NGN",
		Balance:         transactionsResp.TotalAmountReceived, // Total from transactions
		AvailableBalance: transactionsResp.TotalAmountReceived, // Available = Total (no locked amount concept in Kora API)
		LastUpdated:     time.Now(),
	}

	return result, nil
}

// ============================================================================
// Transaction Listing (VirtualAccountTransactionLister interface)
// ============================================================================

// ListTransactions implements the VirtualAccountTransactionLister interface.
// Gets transaction history for a virtual account.
func (p *korapayVirtualAccountProvider) ListTransactions(ctx context.Context, accountNumber string, startDate, endDate *string, page, limit int) (*domain.VirtualAccountTransactionsResult, error) {
	// Set defaults
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	// Call Korapay API
	koraResponse, err := p.client.GetVirtualAccountTransactions(accountNumber, startDate, endDate, page, limit)
	if err != nil {
		return nil, fmt.Errorf("korapay get virtual account transactions failed: %w", err)
	}

	// Map response to unified format
	return p.mapTransactionsResponse(accountNumber, koraResponse), nil
}

// mapTransactionsResponse maps Korapay transactions response to unified VirtualAccountTransactionsResult.
func (p *korapayVirtualAccountProvider) mapTransactionsResponse(accountNumber string, resp *VirtualAccountTransactionsResponse) *domain.VirtualAccountTransactionsResult {
	result := &domain.VirtualAccountTransactionsResult{
		Provider:            domain.ProviderKorapay,
		AccountNumber:       accountNumber,
		Currency:            "NGN",
		Transactions:        []domain.VirtualAccountTransaction{},
	}

	// Korapay returns status: true/false (boolean) in top level
	if resp.Status && resp.Data != nil {
		result.Success = true
		result.TotalAmountReceived = resp.Data.TotalAmountReceived
		result.Currency = resp.Data.Currency

		// Map transactions
		for _, koraTx := range resp.Data.Transactions {
			// Parse amount from string (e.g., "1000.00") to int64 (kobo)
			amountInKobo, err := p.parseAmountToKobo(koraTx.Amount)
			if err != nil {
				// Skip invalid amounts, log error (for now, we'll continue)
				continue
			}

			// Parse fee from string to int64 (kobo)
			feeInKobo := int64(0)
			if koraTx.Fee != "" {
				if parsed, err := p.parseAmountToKobo(koraTx.Fee); err == nil {
					feeInKobo = parsed
				}
			}

			// Parse processed_at timestamp (Kora might not provide this, use current time as fallback)
			processedAt := time.Now()
			// If Kora provides timestamp in transactions, parse it here

			tx := domain.VirtualAccountTransaction{
				Reference:   koraTx.Reference,
				Status:      koraTx.Status,
				Amount:      amountInKobo,
				Fee:         feeInKobo,
				Currency:    koraTx.Currency,
				Description: koraTx.Description,
				ProcessedAt: processedAt,
			}

			// Map payer bank account if available
			if koraTx.PayerBankAccount != nil {
				tx.PayerBankAccount = &domain.PayerBankAccount{
					AccountNumber: koraTx.PayerBankAccount.AccountNumber,
					AccountName:   koraTx.PayerBankAccount.AccountName,
					BankName:      koraTx.PayerBankAccount.BankName,
				}
			}

			result.Transactions = append(result.Transactions, tx)
		}

		// Map pagination
		if resp.Data.Pagination.Total > 0 {
			result.Pagination = &domain.PaginationInfo{
				Page:       resp.Data.Pagination.Page,
				Total:      resp.Data.Pagination.Total,
				PageCount:  resp.Data.Pagination.PageCount,
				TotalPages: resp.Data.Pagination.TotalPages,
			}
		}
	} else {
		result.Success = false
	}

	return result
}

// parseAmountToKobo parses an amount string (e.g., "1000.00") to int64 (kobo).
func (p *korapayVirtualAccountProvider) parseAmountToKobo(amountStr string) (int64, error) {
	// Parse as float first
	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format: %s", amountStr)
	}

	// Convert to kobo (multiply by 100)
	amountInKobo := int64(amountFloat * 100)
	return amountInKobo, nil
}
