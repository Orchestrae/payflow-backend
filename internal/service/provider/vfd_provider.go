package provider

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/platform/vfd"
)

const (
	ProviderNameVFD     = "vfd"
	ProviderNameKorapay = "korapay"
)

// VFDProvider is an adapter that wraps the VFD service to implement the TransferProvider interface.
type VFDProvider struct {
	vfdService vfd.VFDService
}

// NewVFDProvider creates a new VFD provider adapter.
func NewVFDProvider(vfdService vfd.VFDService) TransferProvider {
	return &VFDProvider{
		vfdService: vfdService,
	}
}

// Name returns the provider identifier.
func (p *VFDProvider) Name() string {
	return ProviderNameVFD
}

// AccountEnquiry implements the TransferProvider interface.
func (p *VFDProvider) AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	return p.vfdService.AccountEnquiry(ctx, accountNumber)
}

// BeneficiaryEnquiry implements the TransferProvider interface.
func (p *VFDProvider) BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	return p.vfdService.BeneficiaryEnquiry(ctx, accountNo, bank, transferType)
}

// GetBankList implements the TransferProvider interface.
func (p *VFDProvider) GetBankList(ctx context.Context) (*domain.BankListResponse, error) {
	return p.vfdService.GetBankList(ctx)
}

// InitiateTransfer implements the TransferProvider interface.
// Note: This method signature includes businessID for compatibility, but VFD API doesn't require it.
// The businessID is used for database records, which will be handled by the service layer.
// This method only handles the API call, not database persistence.
func (p *VFDProvider) InitiateTransfer(ctx context.Context, businessID uint, req *domain.TransferRequest) (*domain.TransferResponse, error) {
	// VFD service doesn't use businessID, so we ignore it here
	// Note: This only handles the API call. Database operations are handled separately.
	return p.vfdService.InitiateTransfer(ctx, req)
}

