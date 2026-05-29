package platform

import (
	"context"
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// PlatformStats contains platform-wide metrics.
type PlatformStats struct {
	TotalOrganizations  int            `json:"total_organizations"`
	ActiveOrganizations int            `json:"active_organizations"`
	SuspendedOrgs       int            `json:"suspended_organizations"`
	TotalEmployees      int            `json:"total_employees"`
	MRR                 int64          `json:"mrr"`
	SignupsThisMonth    int            `json:"signups_this_month"`
	PlanDistribution    map[string]int `json:"plan_distribution"`
}

// OrgSummary contains a business summary for the platform admin.
type OrgSummary struct {
	BusinessID         uint      `json:"business_id"`
	BusinessName       string    `json:"business_name"`
	AdminEmail         string    `json:"admin_email"`
	EmployeeCount      int       `json:"employee_count"`
	PlanTier           string    `json:"plan_tier"`
	SubscriptionStatus string    `json:"subscription_status"`
	IsSuspended        bool      `json:"is_suspended"`
	CreatedAt          time.Time `json:"created_at"`
}

// PlatformService manages platform-wide operations for super admins.
type PlatformService interface {
	GetStats(ctx context.Context) (*PlatformStats, error)
	ListOrganizations(ctx context.Context, page, limit int) ([]*OrgSummary, int, error)
	SuspendOrganization(ctx context.Context, businessID uint, reason string) error
	ActivateOrganization(ctx context.Context, businessID uint) error
}

type platformService struct {
	businessRepo repository.BusinessRepository
	userRepo     repository.UserRepository
	employeeRepo repository.EmployeeRepository
	subRepo      repository.SubscriptionRepository
	planRepo     repository.SubscriptionPlanRepository
}

func NewPlatformService(
	businessRepo repository.BusinessRepository,
	userRepo repository.UserRepository,
	employeeRepo repository.EmployeeRepository,
	subRepo repository.SubscriptionRepository,
	planRepo repository.SubscriptionPlanRepository,
) PlatformService {
	return &platformService{
		businessRepo: businessRepo,
		userRepo:     userRepo,
		employeeRepo: employeeRepo,
		subRepo:      subRepo,
		planRepo:     planRepo,
	}
}

func (s *platformService) GetStats(ctx context.Context) (*PlatformStats, error) {
	stats := &PlatformStats{
		PlanDistribution: make(map[string]int),
	}

	// Get all subscriptions for MRR and plan distribution
	subs, total, err := s.subRepo.FindAll(ctx, 1, 10000) // Get all
	if err == nil {
		stats.TotalOrganizations = total

		for _, sub := range subs {
			if sub.Status == "active" {
				stats.ActiveOrganizations++
				if sub.Plan != nil {
					stats.MRR += sub.Plan.PriceMonthly
					stats.PlanDistribution[string(sub.Plan.Tier)]++
				}
			}

			// Signups this month
			now := time.Now()
			if sub.CreatedAt.Month() == now.Month() && sub.CreatedAt.Year() == now.Year() {
				stats.SignupsThisMonth++
			}

			// Check suspended
			if sub.Business != nil && sub.Business.IsSuspended {
				stats.SuspendedOrgs++
			}
		}
	}

	return stats, nil
}

func (s *platformService) ListOrganizations(ctx context.Context, page, limit int) ([]*OrgSummary, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	subs, total, err := s.subRepo.FindAll(ctx, page, limit)
	if err != nil {
		return nil, 0, err
	}

	orgs := make([]*OrgSummary, 0, len(subs))
	for _, sub := range subs {
		org := &OrgSummary{
			SubscriptionStatus: sub.Status,
			CreatedAt:          sub.CreatedAt,
		}

		if sub.Business != nil {
			org.BusinessID = sub.Business.ID
			org.BusinessName = sub.Business.Name
			org.PlanTier = string(sub.Business.SubscriptionTier)
			org.IsSuspended = sub.Business.IsSuspended

			// Get admin email
			admin, err := s.userRepo.FindBusinessAdmin(ctx, sub.BusinessID)
			if err == nil {
				org.AdminEmail = admin.Email
			}

			// Get employee count
			employees, err := s.employeeRepo.FindByBusinessID(ctx, sub.BusinessID)
			if err == nil {
				org.EmployeeCount = len(employees)
			}
		}

		if sub.Plan != nil {
			org.PlanTier = string(sub.Plan.Tier)
		}

		orgs = append(orgs, org)
	}

	return orgs, total, nil
}

func (s *platformService) SuspendOrganization(ctx context.Context, businessID uint, reason string) error {
	business, err := s.businessRepo.FindByID(ctx, businessID)
	if err != nil {
		return domain.ErrNotFound
	}

	business.IsSuspended = true
	business.SubscriptionStatus = "suspended"
	if err := s.businessRepo.Update(ctx, business); err != nil {
		return fmt.Errorf("failed to suspend organization: %w", err)
	}
	return nil
}

func (s *platformService) ActivateOrganization(ctx context.Context, businessID uint) error {
	business, err := s.businessRepo.FindByID(ctx, businessID)
	if err != nil {
		return domain.ErrNotFound
	}

	business.IsSuspended = false
	business.SubscriptionStatus = "active"
	if err := s.businessRepo.Update(ctx, business); err != nil {
		return fmt.Errorf("failed to activate organization: %w", err)
	}
	return nil
}
