// internal/service/auth_service.go
package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/platform/vfd"
	"payflow/internal/repository"
	"payflow/pkg/utils"
	"time"
)

type authService struct {
	userRepo     repository.UserRepository
	businessRepo repository.BusinessRepository
	txer         repository.Transactioner // Transaction manager
	jwtSecret    string
	jwtExpiry    time.Duration
	vfdService   vfd.VFDService
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	userRepo repository.UserRepository,
	businessRepo repository.BusinessRepository,
	txer repository.Transactioner,
	jwtSecret string,
	jwtExpiry time.Duration,
	vfdService vfd.VFDService,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		businessRepo: businessRepo,
		txer:         txer,
		jwtSecret:    jwtSecret,
		jwtExpiry:    jwtExpiry,
		vfdService:   vfdService,
	}
}

// RegisterBusiness handles the creation of a new business and its admin user.
// This operation MUST be transactional. If creating the user fails, the business creation should be rolled back.
func (s *authService) RegisterBusiness(ctx context.Context, name, email, password, rcNumber string, incorporationDate time.Time, directorBVN string) (*domain.User, *vfd.CorporateAccount, error) {
	// Start a new transaction
	tx := s.txer.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			s.txer.Rollback(tx)
			panic(r)
		}
	}()

	// 1. Hash the password
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		s.txer.Rollback(tx)
		return nil, nil, fmt.Errorf("could not hash password: %w", err)
	}

	// 2. Create the business record first (without an AdminID yet)
	business := &domain.Business{
		Name:              name,
		RCNumber:          &rcNumber,
		IncorporationDate: &incorporationDate,
		DirectorBVN:       &directorBVN,
	}

	// Cast tx to repository.Transactioner because WithTx expects it.
	// We assume tx returned by Begin satisfies this, but it returns interface{}.
	// If the underlying repo expects *gorm.DB, we might need to change how we pass it.
	// But based on repository.go, WithTx takes Transactioner.
	// If txer is properly implemented, tx should be a Transactioner.
	// Wait, if tx is *gorm.DB, it DOES implement Transactioner methods (Begin, Commit, Rollback).
	// So we can assert it.
	txerObj, ok := tx.(repository.Transactioner)
	if !ok {
		// Fallback: If it's *gorm.DB, it technically matches the interface but maybe not explicitly?
		// Actually, *gorm.DB has Begin(), Commit(), Rollback().
		// But WithTx in repository.go might be expecting the interface.
		// Let's try assertion.
		s.txer.Rollback(tx)
		return nil, nil, fmt.Errorf("transaction object does not implement Transactioner interface")
	}

	businessRepoTx := s.businessRepo.WithTx(txerObj)
	if err := businessRepoTx.Create(ctx, business); err != nil {
		s.txer.Rollback(tx)
		if err == domain.ErrConflict {
			return nil, nil, fmt.Errorf("business already exists: %w", domain.ErrConflict)
		}
		return nil, nil, fmt.Errorf("could not create business: %w", err)
	}

	// 3. Create the admin user record
	adminUser := &domain.User{
		BusinessID:   business.ID,
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         domain.RoleAdmin,
		IsVerified:   false, // Requires email verification
	}

	// Use a specific user repository that is aware of the transaction
	userRepoTx := s.userRepo.WithTx(txerObj)
	if err := userRepoTx.Create(ctx, adminUser); err != nil {
		s.txer.Rollback(tx)
		if err == domain.ErrConflict {
			return nil, nil, fmt.Errorf("user with email '%s' already exists: %w", email, domain.ErrConflict)
		}
		return nil, nil, fmt.Errorf("could not create admin user: %w", err)
	}

	// 4. Update the business with the new admin's ID
	business.AdminID = adminUser.ID
	if err := businessRepoTx.Update(ctx, business); err != nil {
		s.txer.Rollback(tx)
		return nil, nil, fmt.Errorf("could not link admin to business: %w", err)
	}

	// 5. Try to create VFD corporate account (optional - may not be configured)
	var corporateAccount *vfd.CorporateAccount
	if s.vfdService != nil {
		vfdDetails := vfd.NewAccountDetails{
			RCNumber:          rcNumber,
			CompanyName:       name,
			IncorporationDate: incorporationDate,
			DirectorBVN:       directorBVN,
		}

		account, err := s.vfdService.CreateNewCorporateAccount(ctx, vfdDetails)
		if err != nil {
			// VFD is optional - log the error but don't fail registration
			fmt.Printf("Warning: VFD corporate account creation failed (non-critical): %v\n", err)
		} else {
			corporateAccount = account
			business.VFDAccountNumber = &corporateAccount.AccountNumber
			business.VFDAccountName = &corporateAccount.AccountName
			if err := businessRepoTx.Update(ctx, business); err != nil {
				s.txer.Rollback(tx)
				return nil, nil, fmt.Errorf("could not update business with VFD account details: %w", err)
			}
		}
	}

	// Ensure we have a non-nil response even if VFD wasn't configured
	if corporateAccount == nil {
		corporateAccount = &vfd.CorporateAccount{
			AccountNumber: "",
			AccountName:   name,
		}
	}

	// 6. Commit the transaction
	if err := s.txer.Commit(tx); err != nil {
		s.txer.Rollback(tx)
		return nil, nil, fmt.Errorf("failed to commit registration transaction: %w", err)
	}

	return adminUser, corporateAccount, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	// 1. Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return "", nil, domain.ErrUnauthorized // Use generic error for security
		}
		return "", nil, err // Internal error
	}

	// 2. Check password
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", nil, domain.ErrUnauthorized
	}

	// TODO: Later, add a check for user.IsVerified

	// 3. Generate JWT
	token, err := utils.GenerateToken(
		fmt.Sprint(user.ID),
		fmt.Sprint(user.BusinessID),
		string(user.Role),
		s.jwtSecret,
		s.jwtExpiry,
	)
	if err != nil {
		return "", nil, fmt.Errorf("could not generate token: %w", err)
	}

	return token, user, nil
}
