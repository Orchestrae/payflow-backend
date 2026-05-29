-- Super admin role
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'super_admin';

-- Subscription plans
CREATE TABLE IF NOT EXISTS subscription_plans (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    name VARCHAR(100) NOT NULL,
    tier VARCHAR(20) NOT NULL UNIQUE,
    price_monthly BIGINT DEFAULT 0,
    max_employees INT DEFAULT 0,
    max_payroll_runs INT DEFAULT 0,
    features TEXT,
    is_active BOOLEAN DEFAULT true,
    paystack_plan_code VARCHAR(100)
);

-- Seed default plans
INSERT INTO subscription_plans (name, tier, price_monthly, max_employees, max_payroll_runs, features)
VALUES
    ('Free', 'free', 0, 5, 2, '{"payroll": true, "paye": true}'),
    ('Starter', 'starter', 1500000, 50, 0, '{"payroll": true, "paye": true, "pension": true, "nhf": true, "reports": true, "csv_import": true}'),
    ('Pro', 'pro', 5000000, 0, 0, '{"payroll": true, "paye": true, "pension": true, "nhf": true, "reports": true, "csv_import": true, "api_access": true, "priority_support": true}')
ON CONFLICT (tier) DO NOTHING;

-- Subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    business_id BIGINT NOT NULL UNIQUE,
    plan_id BIGINT NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    paystack_subscription_code VARCHAR(100),
    paystack_customer_code VARCHAR(100),
    cancelled_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_business ON subscriptions(business_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status) WHERE deleted_at IS NULL;

-- Invoices
CREATE TABLE IF NOT EXISTS invoices (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    business_id BIGINT NOT NULL,
    subscription_id BIGINT NOT NULL,
    amount BIGINT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    paid_at TIMESTAMP,
    paystack_ref VARCHAR(100),
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_invoices_business ON invoices(business_id, created_at DESC) WHERE deleted_at IS NULL;

-- Business billing columns
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS subscription_tier VARCHAR(20) DEFAULT 'free';
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS subscription_status VARCHAR(20) DEFAULT 'active';
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS is_suspended BOOLEAN DEFAULT false;
