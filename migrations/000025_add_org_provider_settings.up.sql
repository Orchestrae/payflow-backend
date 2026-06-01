-- Org-level provider API key overrides
-- Businesses can store their own Paystack/Korapay keys
CREATE TABLE IF NOT EXISTS org_provider_settings (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses(id),
    provider VARCHAR(50) NOT NULL,
    setting_key VARCHAR(100) NOT NULL,
    encrypted_value TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(business_id, provider, setting_key)
);

CREATE INDEX idx_org_provider_settings_business ON org_provider_settings(business_id);
CREATE INDEX idx_org_provider_settings_provider ON org_provider_settings(business_id, provider);
