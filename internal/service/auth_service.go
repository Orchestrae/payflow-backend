// internal/service/auth_service.go
package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
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
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	userRepo repository.UserRepository,
	businessRepo repository.BusinessRepository,
	txer repository.Transactioner,
	jwtSecret string,
	jwtExpiry time.Duration,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		businessRepo: businessRepo,
		txer:         txer,
		jwtSecret:    jwtSecret,
		jwtExpiry:    jwtExpiry,
	}
}

// RegisterBusiness handles the creation of a new business and its admin user.
// This operation MUST be transactional. If creating the user fails, the business creation should be rolled back.
func (s *authService) RegisterBusiness(ctx context.Context, name, email, password string) (*domain.User, error) {
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
		return nil, fmt.Errorf("could not hash password: %w", err)
	}

	// 2. Create the business record first (without an AdminID yet)
	business := &domain.Business{Name: name}
	if err := s.businessRepo.Create(ctx, business); err != nil {
		s.txer.Rollback(tx)
		if err == domain.ErrConflict { // Assuming repo can detect this
			return nil, err
		}
		return nil, fmt.Errorf("could not create business: %w", err)
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
	userRepoTx := s.userRepo.WithTx(tx) // We need to add this method to our repo interface
	if err := userRepoTx.Create(ctx, adminUser); err != nil {
		s.txer.Rollback(tx)
		if err == domain.ErrConflict {
			return nil, fmt.Errorf("user with email '%s' already exists", email)
		}
		return nil, fmt.Errorf("could not create admin user: %w", err)
	}

	// 4. Update the business with the new admin's ID
	business.AdminID = adminUser.ID
	if err := s.businessRepo.Update(ctx, business); err != nil { // Need to add Update to repo
		s.txer.Rollback(tx)
		return nil, fmt.Errorf("could not link admin to business: %w", err)
	}

	// 5. Commit the transaction
	if err := s.txer.Commit(tx); err != nil {
		s.txer.Rollback(tx)
		return nil, fmt.Errorf("failed to commit registration transaction: %w", err)
	}

	// TODO: Trigger email verification flow via NotificationService
	// s.notificationService.SendVerificationEmail(ctx, adminUser)

	return adminUser, nil
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
