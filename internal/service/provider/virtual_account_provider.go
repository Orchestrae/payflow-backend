package provider

import (
	"context"
	"payflow/internal/domain"
)

// VirtualAccountProvider defines the core interface for all virtual account providers.
// All provider implementations (Korapay, VFD, etc.) must implement this interface.
// The interface is intentionally minimal - providers implement what they support.
type VirtualAccountProvider interface {
	// Name returns the identifier of the provider (e.g., "korapay", "vfd")
	Name() domain.ProviderName

	// CreateVirtualAccount creates a virtual account for a business.
	// Each provider is responsible for mapping the unified request to its specific API format.
	CreateVirtualAccount(ctx context.Context, req *domain.CreateVirtualAccountRequest) (*domain.VirtualAccountResult, error)

	// GetVirtualAccount retrieves virtual account details by reference.
	GetVirtualAccount(ctx context.Context, accountReference string) (*domain.VirtualAccountResult, error)
}

// VirtualAccountBalancer is an optional interface for providers that support balance checking.
// Not all providers may support this (e.g., if they only provide transaction lists).
type VirtualAccountBalancer interface {
	// GetBalance gets the current balance for a virtual account.
	GetBalance(ctx context.Context, accountReference string) (*domain.VirtualAccountBalanceResult, error)
}

// VirtualAccountTransactionLister is an optional interface for providers that support transaction listing.
// Providers can list deposits/transactions on a virtual account.
type VirtualAccountTransactionLister interface {
	// ListTransactions gets transaction history for a virtual account.
	ListTransactions(ctx context.Context, accountNumber string, startDate, endDate *string, page, limit int) (*domain.VirtualAccountTransactionsResult, error)
}

// DepositWebhookVerifier is an optional interface for providers that send deposit webhooks.
// Providers can verify webhook payloads for security.
type DepositWebhookVerifier interface {
	// VerifyDepositWebhook verifies and parses a deposit webhook payload from the provider.
	// Returns nil if webhook is invalid or cannot be verified.
	VerifyDepositWebhook(ctx context.Context, payload []byte, signature string) (*domain.DepositNotification, error)
}

// VirtualAccountProviderManager manages virtual account providers (similar to TransferProviderManager).
// This allows for provider selection and fallback logic if needed in the future.
type VirtualAccountProviderManager struct {
	defaultProvider VirtualAccountProvider
	allProviders    map[domain.ProviderName]VirtualAccountProvider
}

