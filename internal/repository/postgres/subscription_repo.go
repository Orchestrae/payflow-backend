package postgres

import (
	"context"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// --- SubscriptionPlan Repository ---

type subscriptionPlanRepository struct{ db *gorm.DB }

func NewSubscriptionPlanRepository(db *gorm.DB) repository.SubscriptionPlanRepository {
	return &subscriptionPlanRepository{db: db}
}

func (r *subscriptionPlanRepository) Create(ctx context.Context, plan *domain.SubscriptionPlan) error {
	return r.db.WithContext(ctx).Create(plan).Error
}

func (r *subscriptionPlanRepository) FindAll(ctx context.Context) ([]*domain.SubscriptionPlan, error) {
	var plans []*domain.SubscriptionPlan
	err := r.db.WithContext(ctx).Where("is_active = true").Order("price_monthly ASC").Find(&plans).Error
	return plans, err
}

func (r *subscriptionPlanRepository) FindByTier(ctx context.Context, tier domain.PlanTier) (*domain.SubscriptionPlan, error) {
	var plan domain.SubscriptionPlan
	if err := r.db.WithContext(ctx).Where("tier = ?", tier).First(&plan).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return &plan, nil
}

func (r *subscriptionPlanRepository) FindByID(ctx context.Context, id uint) (*domain.SubscriptionPlan, error) {
	var plan domain.SubscriptionPlan
	if err := r.db.WithContext(ctx).First(&plan, id).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return &plan, nil
}

// --- Subscription Repository ---

type subscriptionRepository struct{ db *gorm.DB }

func NewSubscriptionRepository(db *gorm.DB) repository.SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(ctx context.Context, sub *domain.Subscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *subscriptionRepository) FindByBusinessID(ctx context.Context, businessID uint) (*domain.Subscription, error) {
	var sub domain.Subscription
	if err := r.db.WithContext(ctx).Preload("Plan").Where("business_id = ?", businessID).First(&sub).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return &sub, nil
}

func (r *subscriptionRepository) Update(ctx context.Context, sub *domain.Subscription) error {
	return r.db.WithContext(ctx).Save(sub).Error
}

func (r *subscriptionRepository) FindAll(ctx context.Context, page, limit int) ([]*domain.Subscription, int, error) {
	var subs []*domain.Subscription
	var total int64

	r.db.WithContext(ctx).Model(&domain.Subscription{}).Count(&total)

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).Preload("Plan").Preload("Business").
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&subs).Error

	return subs, int(total), err
}

// --- Invoice Repository ---

type invoiceRepository struct{ db *gorm.DB }

func NewInvoiceRepository(db *gorm.DB) repository.InvoiceRepository {
	return &invoiceRepository{db: db}
}

func (r *invoiceRepository) Create(ctx context.Context, inv *domain.Invoice) error {
	return r.db.WithContext(ctx).Create(inv).Error
}

func (r *invoiceRepository) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.Invoice, int, error) {
	var invoices []*domain.Invoice
	var total int64

	r.db.WithContext(ctx).Model(&domain.Invoice{}).Where("business_id = ?", businessID).Count(&total)

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).Where("business_id = ?", businessID).
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&invoices).Error

	return invoices, int(total), err
}

func (r *invoiceRepository) Update(ctx context.Context, inv *domain.Invoice) error {
	return r.db.WithContext(ctx).Save(inv).Error
}
