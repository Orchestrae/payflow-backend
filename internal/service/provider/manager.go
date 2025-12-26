package provider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"payflow/internal/domain"
)

// TransferProviderManager manages multiple transfer providers and implements
// fallback logic when providers fail.
type TransferProviderManager struct {
	defaultProvider TransferProvider
	fallbackProviders []TransferProvider
	allProviders map[string]TransferProvider
}

// NewTransferProviderManager creates a new provider manager.
// defaultProviderName is the name of the default provider (e.g., "vfd")
// fallbackOrder is a comma-separated list of provider names for fallback (e.g., "korapay,vfd")
// providers is a map of all available providers by name
func NewTransferProviderManager(
	defaultProviderName string,
	fallbackOrder string,
	providers map[string]TransferProvider,
) (*TransferProviderManager, error) {
	defaultProvider, exists := providers[defaultProviderName]
	if !exists {
		return nil, fmt.Errorf("default provider '%s' not found in available providers", defaultProviderName)
	}

	// Parse fallback order
	var fallbackProviders []TransferProvider
	if fallbackOrder != "" {
		fallbackNames := strings.Split(strings.TrimSpace(fallbackOrder), ",")
		for _, name := range fallbackNames {
			name = strings.TrimSpace(name)
			if name == defaultProviderName {
				// Skip default provider in fallback list (it's already the default)
				continue
			}
			if provider, exists := providers[name]; exists {
				fallbackProviders = append(fallbackProviders, provider)
			} else {
				slog.Warn("Fallback provider not found, skipping", "provider", name)
			}
		}
	}

	return &TransferProviderManager{
		defaultProvider:   defaultProvider,
		fallbackProviders: fallbackProviders,
		allProviders:      providers,
	}, nil
}

// GetDefaultProvider returns the default provider.
func (m *TransferProviderManager) GetDefaultProvider() TransferProvider {
	return m.defaultProvider
}

// AccountEnquiry tries the default provider, then fallback providers until one succeeds.
func (m *TransferProviderManager) AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	var lastErr error
	providersToTry := append([]TransferProvider{m.defaultProvider}, m.fallbackProviders...)

	for _, provider := range providersToTry {
		slog.Info("Attempting account enquiry", "provider", provider.Name(), "account", accountNumber)
		response, err := provider.AccountEnquiry(ctx, accountNumber)
		if err == nil {
			slog.Info("Account enquiry succeeded", "provider", provider.Name())
			return response, nil
		}
		lastErr = fmt.Errorf("%s: %w", provider.Name(), err)
		slog.Warn("Account enquiry failed, trying next provider", "provider", provider.Name(), "error", err)
	}

	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

// BeneficiaryEnquiry tries the default provider, then fallback providers until one succeeds.
func (m *TransferProviderManager) BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	var lastErr error
	providersToTry := append([]TransferProvider{m.defaultProvider}, m.fallbackProviders...)

	for _, provider := range providersToTry {
		slog.Info("Attempting beneficiary enquiry", "provider", provider.Name(), "account", accountNo)
		response, err := provider.BeneficiaryEnquiry(ctx, accountNo, bank, transferType)
		if err == nil {
			slog.Info("Beneficiary enquiry succeeded", "provider", provider.Name())
			return response, nil
		}
		lastErr = fmt.Errorf("%s: %w", provider.Name(), err)
		slog.Warn("Beneficiary enquiry failed, trying next provider", "provider", provider.Name(), "error", err)
	}

	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

// GetBankList tries the default provider, then fallback providers until one succeeds.
func (m *TransferProviderManager) GetBankList(ctx context.Context) (*domain.BankListResponse, error) {
	var lastErr error
	providersToTry := append([]TransferProvider{m.defaultProvider}, m.fallbackProviders...)

	for _, provider := range providersToTry {
		slog.Info("Attempting to get bank list", "provider", provider.Name())
		response, err := provider.GetBankList(ctx)
		if err == nil {
			slog.Info("Get bank list succeeded", "provider", provider.Name())
			return response, nil
		}
		lastErr = fmt.Errorf("%s: %w", provider.Name(), err)
		slog.Warn("Get bank list failed, trying next provider", "provider", provider.Name(), "error", err)
	}

	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

// InitiateTransfer tries the default provider, then fallback providers until one succeeds.
func (m *TransferProviderManager) InitiateTransfer(ctx context.Context, businessID uint, req *domain.TransferRequest) (*domain.TransferResponse, error) {
	var lastErr error
	providersToTry := append([]TransferProvider{m.defaultProvider}, m.fallbackProviders...)

	for _, provider := range providersToTry {
		slog.Info("Attempting transfer", "provider", provider.Name(), "reference", req.Reference)
		response, err := provider.InitiateTransfer(ctx, businessID, req)
		if err == nil {
			slog.Info("Transfer succeeded", "provider", provider.Name(), "reference", req.Reference)
			return response, nil
		}
		lastErr = fmt.Errorf("%s: %w", provider.Name(), err)
		slog.Warn("Transfer failed, trying next provider", "provider", provider.Name(), "error", err, "reference", req.Reference)
	}

	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

