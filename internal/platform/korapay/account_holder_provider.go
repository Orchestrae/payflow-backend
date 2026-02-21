package korapay

import (
	"context"
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/internal/service/provider"
)

// Ensure korapayAccountHolderProvider implements required interfaces
var (
	_ provider.AccountHolderProvider = (*korapayAccountHolderProvider)(nil)
)

// korapayAccountHolderProvider implements the AccountHolderProvider interface for Korapay.
type korapayAccountHolderProvider struct {
	client *Client
}

// NewAccountHolderProvider creates a new Korapay account holder provider.
func NewAccountHolderProvider(client *Client) *korapayAccountHolderProvider {
	return &korapayAccountHolderProvider{
		client: client,
	}
}

// Name returns the provider identifier.
func (p *korapayAccountHolderProvider) Name() domain.ProviderName {
	return domain.ProviderKorapay
}

// CreateAccountHolder implements the AccountHolderProvider interface.
// Maps the unified CreateAccountHolderRequest to Korapay's account holder API format.
func (p *korapayAccountHolderProvider) CreateAccountHolder(ctx context.Context, req *domain.CreateAccountHolderRequest) (*domain.AccountHolderResult, error) {
	// Convert domain request to KoraPay format
	koraReq := AccountHolderCreateRequest{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		UseCase:        req.UseCase,
		Type:           req.Type,
		DateOfBirth:    req.DateOfBirth,
		Nationality:    req.Nationality,
		Occupation:     req.Occupation,
		Email:          req.Email,
		Phone:          req.Phone,
		BankIDNumber:   req.BankIDNumber,
		SourceOfInflow: req.SourceOfInflow,
		Metadata:       req.Metadata,
	}

	if req.SourceOfInflowDocument != nil {
		koraReq.SourceOfInflowDocument = &FileReference{
			Reference: req.SourceOfInflowDocument.Reference,
		}
	}

	if req.Selfie != nil {
		koraReq.Selfie = &FileReference{
			Reference: req.Selfie.Reference,
		}
	}

	if req.Identification != nil {
		koraReq.Identification = &AccountHolderIdentification{
			Type:       req.Identification.Type,
			Number:     req.Identification.Number,
			IssuedDate: req.Identification.IssuedDate,
			ExpiryDate: req.Identification.ExpiryDate,
			Country:    req.Identification.Country,
		}
		if req.Identification.DocumentFront != nil {
			koraReq.Identification.DocumentFront = &FileReference{
				Reference: req.Identification.DocumentFront.Reference,
			}
		}
		if req.Identification.DocumentBack != nil {
			koraReq.Identification.DocumentBack = &FileReference{
				Reference: req.Identification.DocumentBack.Reference,
			}
		}
	}

	if req.ProofOfAddress != nil {
		koraReq.ProofOfAddress = &AccountHolderProofOfAddress{
			Type: req.ProofOfAddress.Type,
		}
		if req.ProofOfAddress.Document != nil {
			koraReq.ProofOfAddress.Document = &FileReference{
				Reference: req.ProofOfAddress.Document.Reference,
			}
		}
	}

	if req.Address != nil {
		koraReq.Address = &AccountHolderAddress{
			Country: req.Address.Country,
			Zip:     req.Address.Zip,
			Address: req.Address.Address,
			State:   req.Address.State,
			City:    req.Address.City,
		}
	}

	if req.Employment != nil {
		koraReq.Employment = &AccountHolderEmployment{
			Status:      req.Employment.Status,
			Employer:    req.Employment.Employer,
			Description: req.Employment.Description,
		}
	}

	// Call KoraPay API
	koraResponse, err := p.client.CreateAccountHolder(ctx, koraReq)
	if err != nil {
		return nil, fmt.Errorf("korapay create account holder failed: %w", err)
	}

	if !koraResponse.Status {
		return nil, fmt.Errorf("korapay create account holder error: %s", koraResponse.Message)
	}

	if koraResponse.Data == nil {
		return nil, fmt.Errorf("korapay create account holder returned nil data")
	}

	// Map KoraPay response to domain model
	return &domain.AccountHolderResult{
		Reference: koraResponse.Data.Reference,
		Email:     koraResponse.Data.Email,
		Status:    koraResponse.Data.Status,
		Metadata:  koraResponse.Data.Metadata,
	}, nil
}

// GetAccountHolderDetails implements the AccountHolderProvider interface.
func (p *korapayAccountHolderProvider) GetAccountHolderDetails(ctx context.Context, reference string) (*domain.AccountHolderDetails, error) {
	koraResponse, err := p.client.GetAccountHolderDetails(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("korapay get account holder details failed: %w", err)
	}

	if !koraResponse.Status {
		return nil, fmt.Errorf("korapay get account holder details error: %s", koraResponse.Message)
	}

	if koraResponse.Data == nil {
		return nil, fmt.Errorf("korapay get account holder details returned nil data")
	}

	// Parse date_of_birth
	var dateOfBirth *time.Time
	if koraResponse.Data.DateOfBirth != "" {
		if parsed, err := time.Parse(time.RFC3339, koraResponse.Data.DateOfBirth); err == nil {
			dateOfBirth = &parsed
		}
	}

	// Parse date_created
	var dateCreated time.Time
	if koraResponse.Data.DateCreated != "" {
		if parsed, err := time.Parse(time.RFC3339, koraResponse.Data.DateCreated); err == nil {
			dateCreated = parsed
		} else {
			dateCreated = time.Now() // Fallback
		}
	}

	// Map address
	var address *domain.AccountHolderAddress
	if koraResponse.Data.Address != nil {
		address = &domain.AccountHolderAddress{
			Country: koraResponse.Data.Address.Country,
			Zip:     koraResponse.Data.Address.Zip,
			Address: koraResponse.Data.Address.Address,
			State:   koraResponse.Data.Address.State,
			City:    koraResponse.Data.Address.City,
		}
	}

	// Map documents
	var documents *domain.AccountHolderDocuments
	if koraResponse.Data.Documents != nil {
		documents = &domain.AccountHolderDocuments{
			IdentificationFront: koraResponse.Data.Documents.IdentificationFront,
			IdentificationBack:  koraResponse.Data.Documents.IdentificationBack,
			ProofOfAddress:      koraResponse.Data.Documents.ProofOfAddress,
			Selfie:              koraResponse.Data.Documents.Selfie,
			SourceOfInflow:      koraResponse.Data.Documents.SourceOfInflow,
		}
	}

	return &domain.AccountHolderDetails{
		Reference:      koraResponse.Data.Reference,
		AccountType:    koraResponse.Data.AccountType,
		FirstName:      koraResponse.Data.FirstName,
		LastName:       koraResponse.Data.LastName,
		Email:          koraResponse.Data.Email,
		PhoneNumber:    koraResponse.Data.PhoneNumber,
		Occupation:     koraResponse.Data.Occupation,
		Status:         koraResponse.Data.Status,
		Metadata:       koraResponse.Data.Metadata,
		DateCreated:    dateCreated,
		Country:        koraResponse.Data.Country,
		DateOfBirth:    dateOfBirth,
		Address:        address,
		Documents:      documents,
	}, nil
}

// UpdateAccountHolderKYC implements the AccountHolderProvider interface.
func (p *korapayAccountHolderProvider) UpdateAccountHolderKYC(ctx context.Context, reference string, req *domain.UpdateAccountHolderKYCRequest) (*domain.UpdateAccountHolderKYCResult, error) {
	// Convert domain request to KoraPay format
	koraReq := AccountHolderUpdateKYCRequest{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		SourceOfInflow: req.SourceOfInflow,
	}

	if req.SourceOfInflowDocument != nil {
		koraReq.SourceOfInflowDocument = &FileReference{
			Reference: req.SourceOfInflowDocument.Reference,
		}
	}

	if req.Selfie != nil {
		koraReq.Selfie = &FileReference{
			Reference: req.Selfie.Reference,
		}
	}

	if req.Identification != nil {
		koraReq.Identification = &AccountHolderIdentification{
			Type:       req.Identification.Type,
			Number:     req.Identification.Number,
			IssuedDate: req.Identification.IssuedDate,
			ExpiryDate: req.Identification.ExpiryDate,
			Country:    req.Identification.Country,
		}
		if req.Identification.DocumentFront != nil {
			koraReq.Identification.DocumentFront = &FileReference{
				Reference: req.Identification.DocumentFront.Reference,
			}
		}
		if req.Identification.DocumentBack != nil {
			koraReq.Identification.DocumentBack = &FileReference{
				Reference: req.Identification.DocumentBack.Reference,
			}
		}
	}

	if req.ProofOfAddress != nil {
		koraReq.ProofOfAddress = &AccountHolderProofOfAddress{
			Type: req.ProofOfAddress.Type,
		}
		if req.ProofOfAddress.Document != nil {
			koraReq.ProofOfAddress.Document = &FileReference{
				Reference: req.ProofOfAddress.Document.Reference,
			}
		}
	}

	// Call KoraPay API
	koraResponse, err := p.client.UpdateAccountHolderKYC(ctx, reference, koraReq)
	if err != nil {
		return nil, fmt.Errorf("korapay update account holder KYC failed: %w", err)
	}

	if !koraResponse.Status {
		return nil, fmt.Errorf("korapay update account holder KYC error: %s", koraResponse.Message)
	}

	if koraResponse.Data == nil {
		return nil, fmt.Errorf("korapay update account holder KYC returned nil data")
	}

	// Map KoraPay response to domain model
	return &domain.UpdateAccountHolderKYCResult{
		Reference: koraResponse.Data.Reference,
		FirstName: koraResponse.Data.FirstName,
		LastName:  koraResponse.Data.LastName,
		Status:    koraResponse.Data.Status,
	}, nil
}

// GenerateFileUploadURL implements the AccountHolderProvider interface.
func (p *korapayAccountHolderProvider) GenerateFileUploadURL(ctx context.Context, req *domain.GenerateFileUploadURLRequest) (*domain.FileUploadURLResult, error) {
	// Convert domain request to KoraPay format
	koraReq := FileUploadURLRequest{
		Reference:   req.Reference,
		Purpose:     req.Purpose,
		ContentType: req.ContentType,
	}

	// Call KoraPay API
	koraResponse, err := p.client.GenerateFileUploadURL(ctx, koraReq)
	if err != nil {
		return nil, fmt.Errorf("korapay generate file upload URL failed: %w", err)
	}

	if !koraResponse.Status {
		return nil, fmt.Errorf("korapay generate file upload URL error: %s", koraResponse.Message)
	}

	if koraResponse.Data == nil {
		return nil, fmt.Errorf("korapay generate file upload URL returned nil data")
	}

	// Parse expiration time
	var expires time.Time
	if koraResponse.Data.UploadURLExpires != "" {
		if parsed, err := time.Parse(time.RFC3339, koraResponse.Data.UploadURLExpires); err == nil {
			expires = parsed
		} else {
			expires = time.Now().Add(1 * time.Hour) // Fallback: 1 hour from now
		}
	}

	// Map KoraPay response to domain model
	return &domain.FileUploadURLResult{
		KorapayReference: koraResponse.Data.KorapayReference,
		OwnerReference:   koraResponse.Data.OwnerReference,
		Purpose:          koraResponse.Data.Purpose,
		UploadURL:        koraResponse.Data.UploadURL,
		UploadURLExpires: expires,
	}, nil
}
