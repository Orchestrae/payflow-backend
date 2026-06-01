package provider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"payflow/internal/domain"
)

// DefaultConcurrentWorkers is the default number of concurrent workers for bulk transfers
// when using sequential fallback (for providers without native bulk support).
const DefaultConcurrentWorkers = 5

// TransferProviderManager manages multiple transfer providers and implements
// fallback logic when providers fail.
type TransferProviderManager struct {
	defaultProvider   TransferProvider
	fallbackProviders []TransferProvider
	allProviders      map[domain.ProviderName]TransferProvider
	circuitBreakers   *ProviderCircuitBreakers
}

// NewTransferProviderManager creates a new provider manager.
func NewTransferProviderManager(
	defaultProviderName string,
	fallbackOrder string,
	providers map[domain.ProviderName]TransferProvider,
) (*TransferProviderManager, error) {
	defaultProvider, exists := providers[domain.ProviderName(defaultProviderName)]
	if !exists {
		return nil, fmt.Errorf("default provider '%s' not found in available providers", defaultProviderName)
	}

	fallbackProviders := parseFallbackProviders(fallbackOrder, defaultProviderName, providers)

	return &TransferProviderManager{
		defaultProvider:   defaultProvider,
		fallbackProviders: fallbackProviders,
		allProviders:      providers,
		circuitBreakers:   NewProviderCircuitBreakers(DefaultCircuitBreakerConfig()),
	}, nil
}

// parseFallbackProviders parses the comma-separated fallback order string into providers.
func parseFallbackProviders(fallbackOrder, defaultProviderName string, providers map[domain.ProviderName]TransferProvider) []TransferProvider {
	if fallbackOrder == "" {
		return nil
	}

	var result []TransferProvider
	for _, name := range strings.Split(strings.TrimSpace(fallbackOrder), ",") {
		name = strings.TrimSpace(name)
		if name == defaultProviderName {
			continue // Skip default provider - it's already first
		}
		if provider, exists := providers[domain.ProviderName(name)]; exists {
			result = append(result, provider)
		} else {
			slog.Warn("Fallback provider not found, skipping", "provider", name)
		}
	}
	return result
}

// GetDefaultProvider returns the default provider.
func (m *TransferProviderManager) GetDefaultProvider() TransferProvider {
	return m.defaultProvider
}

// GetProvider returns a specific provider by name.
func (m *TransferProviderManager) GetProvider(name domain.ProviderName) (TransferProvider, bool) {
	provider, exists := m.allProviders[name]
	return provider, exists
}

// ============================================================================
// Transfer Operations
// ============================================================================

// InitiateTransfer tries providers in order until one succeeds.
// Order: default → fallbacks (configured via environment)
func (m *TransferProviderManager) InitiateTransfer(ctx context.Context, req *domain.SingleTransferRequest) (*domain.TransferResult, error) {
	providers := m.getDefaultAndFallbackProviders()
	return m.tryTransfer(ctx, providers, req)
}

// tryTransfer attempts the transfer with each provider until one succeeds.
func (m *TransferProviderManager) tryTransfer(ctx context.Context, providers []TransferProvider, req *domain.SingleTransferRequest) (*domain.TransferResult, error) {
	var lastErr error

	for _, provider := range providers {
		result, err := m.attemptTransfer(ctx, provider, req)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

// attemptTransfer tries a single transfer with one provider, respecting circuit breaker.
func (m *TransferProviderManager) attemptTransfer(ctx context.Context, provider TransferProvider, req *domain.SingleTransferRequest) (*domain.TransferResult, error) {
	cb := m.circuitBreakers.Get(provider.Name())

	if !cb.Allow() {
		slog.Warn("Circuit breaker open, skipping provider",
			"provider", provider.Name(),
			"reference", req.Reference,
		)
		return nil, fmt.Errorf("%s: %w", provider.Name(), ErrCircuitOpen)
	}

	slog.Info("Attempting transfer",
		"provider", provider.Name(),
		"reference", req.Reference,
		"amount", req.Amount,
	)

	result, err := provider.InitiateTransfer(ctx, req)
	if err != nil {
		cb.RecordFailure()
		slog.Warn("Transfer failed",
			"provider", provider.Name(),
			"reference", req.Reference,
			"error", err,
		)
		return nil, fmt.Errorf("%s: %w", provider.Name(), err)
	}

	cb.RecordSuccess()
	slog.Info("Transfer succeeded",
		"provider", provider.Name(),
		"reference", req.Reference,
		"status", result.Status,
	)
	return result, nil
}

// ============================================================================
// Bulk Transfer Operations
// ============================================================================

// InitiateBulkTransfer processes multiple transfers as efficiently as possible.
// Strategy:
// 1. If provider supports native bulk AND batch size is valid → use native bulk (single API call)
// 2. If provider has native bulk but batch is too small → use concurrent singles
// 3. If provider doesn't support bulk → use concurrent singles with worker pool
func (m *TransferProviderManager) InitiateBulkTransfer(ctx context.Context, req *domain.BulkTransferRequest) (*domain.BulkTransferResult, error) {
	provider := m.defaultProvider

	// Check if provider supports native bulk transfers
	if bulkProvider, ok := provider.(BulkTransferrer); ok {
		return m.handleBulkWithNativeSupport(ctx, bulkProvider, req)
	}

	// Fallback to concurrent single transfers
	slog.Info("Provider doesn't support native bulk, using concurrent singles",
		"provider", provider.Name(),
		"transfer_count", len(req.Transfers),
	)
	return m.executeBulkAsConcurrentSingles(ctx, provider, req)
}

// handleBulkWithNativeSupport handles bulk transfers for providers with native bulk support.
// Korapay requires 2-50 transfers per batch, so we need to handle edge cases.
func (m *TransferProviderManager) handleBulkWithNativeSupport(ctx context.Context, bulkProvider BulkTransferrer, req *domain.BulkTransferRequest) (*domain.BulkTransferResult, error) {
	transferCount := len(req.Transfers)
	minBatch := bulkProvider.MinBatchSize()
	maxBatch := bulkProvider.MaxBatchSize()

	slog.Info("Using native bulk transfer",
		"provider", bulkProvider.(TransferProvider).Name(),
		"transfer_count", transferCount,
		"min_batch", minBatch,
		"max_batch", maxBatch,
	)

	// If batch is too small for native bulk, use concurrent singles
	if transferCount < minBatch {
		slog.Info("Batch too small for native bulk, using concurrent singles",
			"transfer_count", transferCount,
			"min_required", minBatch,
		)
		return m.executeBulkAsConcurrentSingles(ctx, bulkProvider.(TransferProvider), req)
	}

	// If batch is within limits, use native bulk
	if transferCount <= maxBatch {
		return bulkProvider.InitiateBulkTransfer(ctx, req)
	}

	// If batch exceeds max, split into multiple batches
	return m.splitAndProcessBulk(ctx, bulkProvider, req, maxBatch)
}

// splitAndProcessBulk splits a large batch into smaller chunks and processes them.
func (m *TransferProviderManager) splitAndProcessBulk(ctx context.Context, bulkProvider BulkTransferrer, req *domain.BulkTransferRequest, maxBatch int) (*domain.BulkTransferResult, error) {
	slog.Info("Splitting large batch into chunks",
		"total_transfers", len(req.Transfers),
		"max_per_batch", maxBatch,
	)

	allResults := make([]domain.TransferResult, 0, len(req.Transfers))
	var firstError error

	// Process in chunks
	for i := 0; i < len(req.Transfers); i += maxBatch {
		end := i + maxBatch
		if end > len(req.Transfers) {
			end = len(req.Transfers)
		}

		chunk := req.Transfers[i:end]
		chunkRef := fmt.Sprintf("%s-chunk-%d", req.BatchReference, i/maxBatch+1)

		chunkReq := &domain.BulkTransferRequest{
			BatchReference:    chunkRef,
			Description:       req.Description,
			Currency:          req.Currency,
			MerchantBearsCost: req.MerchantBearsCost,
			Transfers:         chunk,
			BusinessID:        req.BusinessID,
			BusinessEmail:     req.BusinessEmail,
		}

		// If chunk is too small for native bulk, use concurrent singles
		if len(chunk) < bulkProvider.MinBatchSize() {
			result, err := m.executeBulkAsConcurrentSingles(ctx, bulkProvider.(TransferProvider), chunkReq)
			if err != nil && firstError == nil {
				firstError = err
			}
			if result != nil {
				allResults = append(allResults, result.TransferResults...)
			}
			continue
		}

		result, err := bulkProvider.InitiateBulkTransfer(ctx, chunkReq)
		if err != nil {
			if firstError == nil {
				firstError = err
			}
			slog.Warn("Chunk processing failed", "chunk", chunkRef, "error", err)
			continue
		}

		if result.TransferResults != nil {
			allResults = append(allResults, result.TransferResults...)
		}
	}

	// Aggregate results
	return m.aggregateBulkResults(req.BatchReference, bulkProvider.(TransferProvider).Name(), allResults, firstError), nil
}

// executeBulkAsConcurrentSingles processes transfers concurrently using single transfer endpoint.
// Uses a worker pool to limit concurrent API calls and avoid overwhelming the provider.
func (m *TransferProviderManager) executeBulkAsConcurrentSingles(ctx context.Context, provider TransferProvider, req *domain.BulkTransferRequest) (*domain.BulkTransferResult, error) {
	transferCount := len(req.Transfers)

	slog.Info("Processing bulk as concurrent singles",
		"provider", provider.Name(),
		"transfer_count", transferCount,
		"workers", DefaultConcurrentWorkers,
	)

	// Create channels for work distribution
	jobs := make(chan *domain.SingleTransferRequest, transferCount)
	results := make(chan domain.TransferResult, transferCount)

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < DefaultConcurrentWorkers; w++ {
		wg.Add(1)
		go m.transferWorker(ctx, provider, jobs, results, &wg)
	}

	// Send jobs to workers
	for i := range req.Transfers {
		transfer := &req.Transfers[i]
		// Enrich with business context
		transfer.BusinessID = req.BusinessID
		transfer.BusinessEmail = req.BusinessEmail
		if transfer.Currency == "" {
			transfer.Currency = req.Currency
		}
		jobs <- transfer
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Collect results
	allResults := make([]domain.TransferResult, 0, transferCount)
	for result := range results {
		allResults = append(allResults, result)
	}

	return m.aggregateBulkResults(req.BatchReference, provider.Name(), allResults, nil), nil
}

// transferWorker is a worker goroutine that processes single transfers.
func (m *TransferProviderManager) transferWorker(ctx context.Context, provider TransferProvider, jobs <-chan *domain.SingleTransferRequest, results chan<- domain.TransferResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for transfer := range jobs {
		result, err := provider.InitiateTransfer(ctx, transfer)
		if err != nil {
			results <- domain.TransferResult{
				Success:   false,
				Reference: transfer.Reference,
				Provider:  provider.Name(),
				Status:    "failed",
				Message:   err.Error(),
			}
			continue
		}
		results <- *result
	}
}

// aggregateBulkResults combines individual transfer results into a bulk result.
func (m *TransferProviderManager) aggregateBulkResults(batchReference string, provider domain.ProviderName, results []domain.TransferResult, firstError error) *domain.BulkTransferResult {
	successCount := 0
	failedCount := 0

	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failedCount++
		}
	}

	status := "complete"
	if failedCount > 0 && successCount == 0 {
		status = "failed"
	} else if failedCount > 0 {
		status = "partial"
	}

	result := &domain.BulkTransferResult{
		Success:         failedCount == 0,
		BatchReference:  batchReference,
		Provider:        provider,
		Status:          status,
		TransferResults: results,
	}

	if firstError != nil {
		result.Message = firstError.Error()
	} else if failedCount > 0 {
		result.Message = fmt.Sprintf("%d of %d transfers failed", failedCount, len(results))
	} else {
		result.Message = fmt.Sprintf("All %d transfers successful", len(results))
	}

	return result
}

// ============================================================================
// Account Enquiry Operations
// ============================================================================

// AccountEnquiry tries providers that support account enquiry.
func (m *TransferProviderManager) AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	providers := m.getDefaultAndFallbackProviders()

	for _, provider := range providers {
		enquirer, ok := provider.(AccountEnquirer)
		if !ok {
			continue
		}

		response, err := m.attemptAccountEnquiry(ctx, enquirer, accountNumber)
		if err == nil {
			return response, nil
		}
	}

	return nil, fmt.Errorf("no providers support account enquiry or all failed")
}

func (m *TransferProviderManager) attemptAccountEnquiry(ctx context.Context, enquirer AccountEnquirer, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	slog.Info("Attempting account enquiry", "account", accountNumber)

	response, err := enquirer.AccountEnquiry(ctx, accountNumber)
	if err != nil {
		slog.Warn("Account enquiry failed", "error", err)
		return nil, err
	}

	slog.Info("Account enquiry succeeded")
	return response, nil
}

// ============================================================================
// Beneficiary Enquiry Operations
// ============================================================================

// BeneficiaryEnquiry tries providers that support beneficiary enquiry.
func (m *TransferProviderManager) BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	providers := m.getDefaultAndFallbackProviders()

	for _, provider := range providers {
		enquirer, ok := provider.(BeneficiaryEnquirer)
		if !ok {
			continue
		}

		response, err := m.attemptBeneficiaryEnquiry(ctx, enquirer, accountNo, bank, transferType)
		if err == nil {
			return response, nil
		}
	}

	return nil, fmt.Errorf("no providers support beneficiary enquiry or all failed")
}

func (m *TransferProviderManager) attemptBeneficiaryEnquiry(ctx context.Context, enquirer BeneficiaryEnquirer, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	slog.Info("Attempting beneficiary enquiry", "account", accountNo, "bank", bank)

	response, err := enquirer.BeneficiaryEnquiry(ctx, accountNo, bank, transferType)
	if err != nil {
		slog.Warn("Beneficiary enquiry failed", "error", err)
		return nil, err
	}

	slog.Info("Beneficiary enquiry succeeded")
	return response, nil
}

// ============================================================================
// Bank List Operations
// ============================================================================

// GetBankList tries providers that support bank listing.
func (m *TransferProviderManager) GetBankList(ctx context.Context) (*domain.BankListResponse, error) {
	providers := m.getDefaultAndFallbackProviders()

	for _, provider := range providers {
		lister, ok := provider.(BankLister)
		if !ok {
			continue
		}

		response, err := m.attemptGetBankList(ctx, lister)
		if err == nil {
			return response, nil
		}
	}

	return nil, fmt.Errorf("no providers support bank listing or all failed")
}

func (m *TransferProviderManager) attemptGetBankList(ctx context.Context, lister BankLister) (*domain.BankListResponse, error) {
	slog.Info("Attempting to get bank list")

	response, err := lister.GetBankList(ctx)
	if err != nil {
		slog.Warn("Get bank list failed", "error", err)
		return nil, err
	}

	slog.Info("Get bank list succeeded")
	return response, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// getDefaultAndFallbackProviders returns default + fallback providers.
func (m *TransferProviderManager) getDefaultAndFallbackProviders() []TransferProvider {
	providers := make([]TransferProvider, 0, 1+len(m.fallbackProviders))
	providers = append(providers, m.defaultProvider)
	providers = append(providers, m.fallbackProviders...)
	return providers
}
