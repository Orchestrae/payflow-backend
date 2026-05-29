package domain

import "time"

// PlanTier represents the subscription tier.
type PlanTier string

const (
	PlanFree    PlanTier = "free"
	PlanStarter PlanTier = "starter"
	PlanPro     PlanTier = "pro"
)

// SubscriptionPlan defines a billing plan with limits and pricing.
type SubscriptionPlan struct {
	Model
	Name             string   `gorm:"size:100" json:"name"`
	Tier             PlanTier `gorm:"type:varchar(20);uniqueIndex" json:"tier"`
	PriceMonthly     int64    `json:"price_monthly"`      // kobo
	MaxEmployees     int      `json:"max_employees"`       // 0 = unlimited
	MaxPayrollRuns   int      `json:"max_payroll_runs"`    // per month, 0 = unlimited
	Features         string   `gorm:"type:text" json:"features"` // JSON feature list
	IsActive         bool     `gorm:"default:true" json:"is_active"`
	PaystackPlanCode string   `gorm:"size:100" json:"paystack_plan_code,omitempty"`
}

// Subscription links a business to a plan with billing state.
type Subscription struct {
	Model
	BusinessID               uint       `gorm:"uniqueIndex" json:"business_id"`
	PlanID                   uint       `gorm:"index" json:"plan_id"`
	Status                   string     `gorm:"size:20;default:'active'" json:"status"` // active, past_due, cancelled, suspended
	CurrentPeriodStart       time.Time  `json:"current_period_start"`
	CurrentPeriodEnd         time.Time  `json:"current_period_end"`
	PaystackSubscriptionCode string    `gorm:"size:100" json:"paystack_subscription_code,omitempty"`
	PaystackCustomerCode     string    `gorm:"size:100" json:"paystack_customer_code,omitempty"`
	CancelledAt              *time.Time `json:"cancelled_at,omitempty"`

	Business *Business         `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	Plan     *SubscriptionPlan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}

// Invoice tracks a billing payment.
type Invoice struct {
	Model
	BusinessID     uint       `gorm:"index" json:"business_id"`
	SubscriptionID uint       `json:"subscription_id"`
	Amount         int64      `json:"amount"` // kobo
	Status         string     `gorm:"size:20;default:'pending'" json:"status"` // paid, pending, failed
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	PaystackRef    string     `gorm:"size:100" json:"paystack_ref,omitempty"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
}
