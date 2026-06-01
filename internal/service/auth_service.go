// internal/service/auth_service.go
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/platform/vfd"
	"payflow/internal/repository"
	"payflow/pkg/utils"
	"time"

	"github.com/rs/zerolog/log"
)

type authService struct {
	userRepo        repository.UserRepository
	businessRepo    repository.BusinessRepository
	cadreRepo       repository.CadreRepository
	employeeRepo    repository.EmployeeRepository
	planRepo        repository.SubscriptionPlanRepository
	subRepo         repository.SubscriptionRepository
	txer            repository.Transactioner
	jwtSecret       string
	jwtExpiry       time.Duration
	vfdService      vfd.VFDService
	notificationSvc NotificationService
	appURL          string
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	userRepo repository.UserRepository,
	businessRepo repository.BusinessRepository,
	txer repository.Transactioner,
	jwtSecret string,
	jwtExpiry time.Duration,
	vfdService vfd.VFDService,
	opts ...AuthServiceOption,
) AuthService {
	svc := &authService{
		userRepo:     userRepo,
		businessRepo: businessRepo,
		txer:         txer,
		jwtSecret:    jwtSecret,
		jwtExpiry:    jwtExpiry,
		vfdService:   vfdService,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// AuthServiceOption configures optional auth service dependencies.
type AuthServiceOption func(*authService)

// WithNotificationService sets the notification service for auth emails.
func WithNotificationService(ns NotificationService, appURL string) AuthServiceOption {
	return func(s *authService) {
		s.notificationSvc = ns
		s.appURL = appURL
	}
}

// WithCadreRepo sets the cadre repository for default cadre creation on registration.
func WithCadreRepo(repo repository.CadreRepository) AuthServiceOption {
	return func(s *authService) {
		s.cadreRepo = repo
	}
}

// WithSubscriptionRepos sets repos for auto-assigning Free plan on registration.
func WithSubscriptionRepos(planRepo repository.SubscriptionPlanRepository, subRepo repository.SubscriptionRepository) AuthServiceOption {
	return func(s *authService) {
		s.planRepo = planRepo
		s.subRepo = subRepo
	}
}

// WithEmployeeRepo sets the employee repository for employee self-service login.
func WithEmployeeRepo(repo repository.EmployeeRepository) AuthServiceOption {
	return func(s *authService) {
		s.employeeRepo = repo
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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

	// 5. Store BVN securely (last 4 only for display)
	if len(directorBVN) >= 4 {
		business.DirectorBVNLast4 = directorBVN[len(directorBVN)-4:]
	}
	businessRepoTx.Update(ctx, business)

	// 6. Auto-create default cadre with standard Nigerian salary components
	if s.cadreRepo != nil {
		defaultCadre := &domain.Cadre{
			BusinessID: business.ID,
			Name:       "Standard",
			EarningComponents: []domain.EarningComponent{
				{Name: "Basic Salary", Amount: 15000000, ComponentType: domain.ComponentBasic},       // NGN 150,000
				{Name: "Housing Allowance", Amount: 5000000, ComponentType: domain.ComponentHousing},  // NGN 50,000
				{Name: "Transport Allowance", Amount: 3000000, ComponentType: domain.ComponentTransport}, // NGN 30,000
			},
		}
		if err := s.cadreRepo.Create(ctx, defaultCadre); err != nil {
			log.Warn().Err(err).Msg("Failed to create default cadre — business can create manually")
		} else {
			log.Info().Uint("business_id", business.ID).Msg("Default cadre 'Standard' created")
		}
	}

	// 7. Auto-assign Free subscription plan
	if s.planRepo != nil && s.subRepo != nil {
		freePlan, err := s.planRepo.FindByTier(ctx, domain.PlanFree)
		if err == nil && freePlan != nil {
			now := time.Now()
			sub := &domain.Subscription{
				BusinessID:         business.ID,
				PlanID:             freePlan.ID,
				Status:             "active",
				CurrentPeriodStart: now,
				CurrentPeriodEnd:   now.AddDate(100, 0, 0), // Free = never expires
			}
			if err := s.subRepo.Create(ctx, sub); err != nil {
				log.Warn().Err(err).Msg("Failed to assign Free plan — business can subscribe manually")
			} else {
				business.SubscriptionTier = domain.PlanFree
				business.SubscriptionStatus = "active"
				businessRepoTx.Update(ctx, business)
				log.Info().Uint("business_id", business.ID).Msg("Free plan assigned")
			}
		}
	}

	// 8. Try to create VFD corporate account (optional - may not be configured)
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

	// Send verification email (non-blocking)
	go s.SendVerificationEmail(context.Background(), adminUser.ID)

	// Send welcome email (non-blocking)
	if s.notificationSvc != nil {
		go s.notificationSvc.SendEmail(context.Background(), email,
			"Welcome to PayFlow!",
			fmt.Sprintf("Hi! Your business '%s' has been registered on PayFlow.\n\n"+
				"We've created a default salary structure ('Standard') to get you started.\n\n"+
				"Next steps:\n"+
				"1. Log in at %s/login\n"+
				"2. Add your employees\n"+
				"3. Run your first payroll\n\n"+
				"Welcome aboard!", name, s.appURL))
	}

	log.Info().Str("email", email).Str("business", name).Msg("Business registered with default onboarding")

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

// InviteUser invites a new user to a business.
func (s *authService) InviteUser(ctx context.Context, businessID uint, email string, role domain.UserRole, businessName string) error {
	// Check if user already exists with this email
	existing, _ := s.userRepo.FindByEmail(ctx, email)
	if existing != nil {
		return domain.ErrConflict
	}

	// Generate invite token
	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("failed to generate invite token: %w", err)
	}

	// Create user with invite token (temporary password, must accept invitation)
	tempHash, _ := utils.HashPassword(token) // temp password = token itself
	user := &domain.User{
		BusinessID:   businessID,
		Email:        email,
		PasswordHash: tempHash,
		Role:         role,
		InviteToken:  &token,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create invited user: %w", err)
	}

	// Send invitation email
	if s.notificationSvc != nil {
		inviteURL := fmt.Sprintf("%s/invite?token=%s", s.appURL, token)
		subject := fmt.Sprintf("You're invited to join %s on PayFlow", businessName)
		body := fmt.Sprintf("You have been invited to join %s as %s.\n\nAccept your invitation: %s\n\nThis link is valid for 7 days.", businessName, role, inviteURL)
		go s.notificationSvc.SendEmail(context.Background(), email, subject, body)
	}

	log.Info().Str("email", email).Str("role", string(role)).Uint("business_id", businessID).Msg("User invited")
	return nil
}

// AcceptInvitation accepts an invitation and sets the user's password.
func (s *authService) AcceptInvitation(ctx context.Context, token string, password string) (*domain.User, string, error) {
	// Find all users for this business and look for matching invite token
	// Since we don't have a FindByInviteToken, iterate users
	// TODO: Add FindByInviteToken to UserRepository for efficiency
	user, err := s.userRepo.FindByInviteToken(ctx, token)
	if err != nil {
		return nil, "", domain.ErrNotFound
	}
	if user.InviteAccepted {
		return nil, "", fmt.Errorf("%w: invitation already accepted", domain.ErrConflict)
	}

	// Set real password
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = hashedPassword
	user.InviteAccepted = true
	user.InviteToken = nil
	user.IsVerified = true

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, "", fmt.Errorf("failed to update user: %w", err)
	}

	// Generate JWT
	jwtToken, err := utils.GenerateToken(
		fmt.Sprint(user.ID),
		fmt.Sprint(user.BusinessID),
		string(user.Role),
		s.jwtSecret,
		s.jwtExpiry,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, jwtToken, nil
}

// RequestPasswordReset sends a password reset email.
func (s *authService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether email exists
		return nil
	}

	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	expiry := time.Now().Add(1 * time.Hour)
	user.ResetToken = &token
	user.ResetTokenExpiry = &expiry

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	// Send reset email
	if s.notificationSvc != nil {
		resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, token)
		subject := "PayFlow - Reset Your Password"
		body := fmt.Sprintf("You requested a password reset.\n\nReset your password: %s\n\nThis link expires in 1 hour.\n\nIf you didn't request this, ignore this email.", resetURL)
		go s.notificationSvc.SendEmail(context.Background(), email, subject, body)
	}

	log.Info().Str("email", email).Msg("Password reset requested")
	return nil
}

// ResetPassword resets the user's password using a valid token.
func (s *authService) ResetPassword(ctx context.Context, token string, newPassword string) error {
	user, err := s.userRepo.FindByResetToken(ctx, token)
	if err != nil {
		return domain.ErrNotFound
	}

	// Check expiry
	if user.ResetTokenExpiry != nil && user.ResetTokenExpiry.Before(time.Now()) {
		return fmt.Errorf("%w: reset token has expired", domain.ErrValidationFailed)
	}

	// Set new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = hashedPassword
	user.ResetToken = nil
	user.ResetTokenExpiry = nil

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	log.Info().Uint("user_id", user.ID).Msg("Password reset completed")
	return nil
}

// SendVerificationEmail generates a token and sends a verification email.
func (s *authService) SendVerificationEmail(ctx context.Context, userID uint) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.IsVerified {
		return nil // Already verified
	}

	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	expiry := time.Now().Add(24 * time.Hour)
	user.EmailVerificationToken = &token
	user.EmailVerificationExpiry = &expiry

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save verification token: %w", err)
	}

	if s.notificationSvc != nil {
		verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, token)
		subject := "PayFlow - Verify Your Email"
		body := fmt.Sprintf("Please verify your email address to complete your registration.\n\nVerify your email: %s\n\nThis link expires in 24 hours.", verifyURL)
		go s.notificationSvc.SendEmail(context.Background(), user.Email, subject, body)
	}

	log.Info().Uint("user_id", userID).Msg("Verification email sent")
	return nil
}

// VerifyEmail verifies a user's email using the verification token.
func (s *authService) VerifyEmail(ctx context.Context, token string) error {
	user, err := s.userRepo.FindByEmailVerificationToken(ctx, token)
	if err != nil {
		return domain.ErrNotFound
	}

	if user.EmailVerificationExpiry != nil && user.EmailVerificationExpiry.Before(time.Now()) {
		return fmt.Errorf("%w: verification link has expired", domain.ErrValidationFailed)
	}

	user.IsVerified = true
	user.EmailVerificationToken = nil
	user.EmailVerificationExpiry = nil

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	log.Info().Uint("user_id", user.ID).Msg("Email verified successfully")
	return nil
}

// CreateEmployeeLogin creates a user account for an employee to enable self-service login.
func (s *authService) CreateEmployeeLogin(ctx context.Context, businessID, employeeID uint, tempPassword string) (*domain.User, error) {
	if s.employeeRepo == nil {
		return nil, fmt.Errorf("employee repository not configured")
	}

	// Verify the employee exists and belongs to the business
	employee, err := s.employeeRepo.FindByID(ctx, employeeID, businessID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	// Check if user already exists for this employee
	existing, _ := s.userRepo.FindByEmail(ctx, employee.Email)
	if existing != nil && existing.EmployeeID != nil && *existing.EmployeeID == employeeID {
		return nil, fmt.Errorf("%w: employee already has a login", domain.ErrConflict)
	}

	// Hash the temporary password
	hashedPassword, err := utils.HashPassword(tempPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user with employee role
	user := &domain.User{
		BusinessID:   businessID,
		Email:        employee.Email,
		PasswordHash: hashedPassword,
		Role:         domain.RoleEmployee,
		EmployeeID:   &employeeID,
		IsVerified:   true, // Admin-created, no verification needed
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if err == domain.ErrConflict {
			return nil, fmt.Errorf("%w: email already in use", domain.ErrConflict)
		}
		return nil, fmt.Errorf("failed to create employee login: %w", err)
	}

	// Link user to employee
	employee.UserID = &user.ID
	if err := s.employeeRepo.Update(ctx, employee); err != nil {
		log.Warn().Err(err).Msg("Failed to link user to employee")
	}

	// Send notification email
	if s.notificationSvc != nil {
		loginURL := fmt.Sprintf("%s/employee-login", s.appURL)
		subject := "Your PayFlow Self-Service Account"
		body := fmt.Sprintf("Hi %s,\n\nA self-service account has been created for you on PayFlow.\n\n"+
			"Login URL: %s\nEmail: %s\nTemporary Password: %s\n\n"+
			"Please change your password after your first login.\n\n"+
			"Best regards,\nPayFlow", employee.FullName, loginURL, employee.Email, tempPassword)
		go s.notificationSvc.SendEmail(context.Background(), employee.Email, subject, body)
	}

	log.Info().Uint("employee_id", employeeID).Uint("user_id", user.ID).Msg("Employee login created")
	return user, nil
}

// EmployeeLogin authenticates an employee user and returns a JWT with employee_id claim.
func (s *authService) EmployeeLogin(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", nil, domain.ErrUnauthorized
	}

	// Must be an employee role
	if user.Role != domain.RoleEmployee {
		return "", nil, domain.ErrUnauthorized
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", nil, domain.ErrUnauthorized
	}

	// Get employee_id
	employeeID := ""
	if user.EmployeeID != nil {
		employeeID = fmt.Sprint(*user.EmployeeID)
	}

	token, err := utils.GenerateTokenWithEmployee(
		fmt.Sprint(user.ID),
		fmt.Sprint(user.BusinessID),
		string(user.Role),
		employeeID,
		s.jwtSecret,
		s.jwtExpiry,
	)
	if err != nil {
		return "", nil, fmt.Errorf("could not generate token: %w", err)
	}

	return token, user, nil
}

