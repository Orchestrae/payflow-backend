package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"payflow/internal/domain"
	"payflow/internal/platform/billing"
	"payflow/internal/repository"
)

// BillingService manages subscriptions and billing.
type BillingService interface {
	GetPlans(ctx context.Context) ([]*domain.SubscriptionPlan, error)
	GetSubscription(ctx context.Context, businessID uint) (*domain.Subscription, error)
	Subscribe(ctx context.Context, businessID uint, tier domain.PlanTier, email, callbackURL string) (paymentURL string, err error)
	CancelSubscription(ctx context.Context, businessID uint) error
	ListInvoices(ctx context.Context, businessID uint, page, limit int) ([]*domain.Invoice, int, error)
	CheckEmployeeLimit(ctx context.Context, businessID uint) error
	AssignFreePlan(ctx context.Context, businessID uint) error
}

type billingService struct {
	planRepo     repository.SubscriptionPlanRepository
	subRepo      repository.SubscriptionRepository
	invoiceRepo  repository.InvoiceRepository
	businessRepo repository.BusinessRepository
	employeeRepo repository.EmployeeRepository
	paystackClient *billing.PaystackBillingClient
	appURL       string
}

func NewBillingService(
	planRepo repository.SubscriptionPlanRepository,
	subRepo repository.SubscriptionRepository,
	invoiceRepo repository.InvoiceRepository,
	businessRepo repository.BusinessRepository,
	employeeRepo repository.EmployeeRepository,
	paystackClient *billing.PaystackBillingClient,
	appURL string,
) BillingService {
	return &billingService{
		planRepo:       planRepo,
		subRepo:        subRepo,
		invoiceRepo:    invoiceRepo,
		businessRepo:   businessRepo,
		employeeRepo:   employeeRepo,
		paystackClient: paystackClient,
		appURL:         appURL,
	}
}

func (s *billingService) GetPlans(ctx context.Context) ([]*domain.SubscriptionPlan, error) {
	return s.planRepo.FindAll(ctx)
}

func (s *billingService) GetSubscription(ctx context.Context, businessID uint) (*domain.Subscription, error) {
	return s.subRepo.FindByBusinessID(ctx, businessID)
}

func (s *billingService) Subscribe(ctx context.Context, businessID uint, tier domain.PlanTier, email, callbackURL string) (string, error) {
	plan, err := s.planRepo.FindByTier(ctx, tier)
	if err != nil {
		return "", fmt.Errorf("plan not found: %w", err)
	}

	// Free plan — just assign directly
	if plan.PriceMonthly == 0 {
		return "", s.AssignFreePlan(ctx, businessID)
	}

	// Paid plan — initialize Paystack transaction
	if s.paystackClient == nil {
		return "", fmt.Errorf("payment provider not configured")
	}

	ref := generateBillingRef()
	if callbackURL == "" {
		callbackURL = fmt.Sprintf("%s/billing/callback", s.appURL)
	}

	paymentURL, err := s.paystackClient.InitializeTransaction(ctx, email, plan.PriceMonthly, ref, plan.PaystackPlanCode, callbackURL)
	if err != nil {
		return "", fmt.Errorf("payment initialization failed: %w", err)
	}

	// Create pending subscription
	now := time.Now()
	sub := &domain.Subscription{
		BusinessID:         businessID,
		PlanID:             plan.ID,
		Status:             "pending",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	if err := s.subRepo.Create(ctx, sub); err != nil {
		log.Warn().Err(err).Msg("Failed to create subscription record")
	}

	// Create pending invoice
	invoice := &domain.Invoice{
		BusinessID:     businessID,
		SubscriptionID: sub.ID,
		Amount:         plan.PriceMonthly,
		Status:         "pending",
		PaystackRef:    ref,
		PeriodStart:    now,
		PeriodEnd:      now.AddDate(0, 1, 0),
	}
	s.invoiceRepo.Create(ctx, invoice)

	// Update business tier
	business, _ := s.businessRepo.FindByID(ctx, businessID)
	if business != nil {
		business.SubscriptionTier = plan.Tier
		business.SubscriptionStatus = "pending"
		s.businessRepo.Update(ctx, business)
	}

	return paymentURL, nil
}

func (s *billingService) CancelSubscription(ctx context.Context, businessID uint) error {
	sub, err := s.subRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		return domain.ErrNotFound
	}

	now := time.Now()
	sub.Status = "cancelled"
	sub.CancelledAt = &now
	if err := s.subRepo.Update(ctx, sub); err != nil {
		return err
	}

	// Downgrade to free
	business, _ := s.businessRepo.FindByID(ctx, businessID)
	if business != nil {
		business.SubscriptionTier = domain.PlanFree
		business.SubscriptionStatus = "active"
		s.businessRepo.Update(ctx, business)
	}

	return nil
}

func (s *billingService) ListInvoices(ctx context.Context, businessID uint, page, limit int) ([]*domain.Invoice, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.invoiceRepo.FindByBusinessID(ctx, businessID, page, limit)
}

func (s *billingService) CheckEmployeeLimit(ctx context.Context, businessID uint) error {
	business, err := s.businessRepo.FindByID(ctx, businessID)
	if err != nil {
		return nil // Can't check, allow
	}

	plan, err := s.planRepo.FindByTier(ctx, business.SubscriptionTier)
	if err != nil || plan.MaxEmployees == 0 {
		return nil // No limit
	}

	employees, err := s.employeeRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		return nil
	}

	if len(employees) >= plan.MaxEmployees {
		return fmt.Errorf("employee limit reached (%d/%d). Upgrade your plan", len(employees), plan.MaxEmployees)
	}
	return nil
}

func (s *billingService) AssignFreePlan(ctx context.Context, businessID uint) error {
	plan, err := s.planRepo.FindByTier(ctx, domain.PlanFree)
	if err != nil {
		return fmt.Errorf("free plan not found: %w", err)
	}

	now := time.Now()
	sub := &domain.Subscription{
		BusinessID:         businessID,
		PlanID:             plan.ID,
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(100, 0, 0), // Free plan never expires
	}

	if err := s.subRepo.Create(ctx, sub); err != nil {
		return fmt.Errorf("failed to create free subscription: %w", err)
	}

	return nil
}

func generateBillingRef() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "PF-INV-" + hex.EncodeToString(b)[:12]
}
