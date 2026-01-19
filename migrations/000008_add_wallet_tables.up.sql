-- Migration: Add business_wallets and wallet_transactions tables
-- Created: 2026-01-14

-- Table: business_wallets
-- Stores virtual account details and balance for each business
CREATE TABLE IF NOT EXISTS business_wallets (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- Business relationship
    business_id INTEGER NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,

    -- Balance tracking (in smallest currency unit, e.g., kobo for NGN)
    balance BIGINT NOT NULL DEFAULT 0, -- Available balance
    locked_balance BIGINT NOT NULL DEFAULT 0, -- Balance locked for pending transfers
    currency VARCHAR(10) NOT NULL DEFAULT 'NGN',
    balance_updated_at TIMESTAMP, -- When balance was last synced from provider

    -- Virtual Account Details (stored from provider after creation)
    virtual_account_number VARCHAR(20) UNIQUE,
    virtual_account_bank_code VARCHAR(10),
    virtual_account_bank_name VARCHAR(255),
    virtual_account_reference VARCHAR(100) UNIQUE, -- Provider's account reference
    virtual_account_unique_id VARCHAR(100), -- Provider's unique ID (e.g., KPY-VA-xxx)
    virtual_account_status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, suspended, inactive

    -- Provider information
    provider VARCHAR(20) NOT NULL, -- korapay, vfd, etc.
    provider_metadata JSONB, -- JSON for provider-specific fields

    -- Indexes
    CONSTRAINT idx_business_wallets_business_id UNIQUE (business_id), -- One wallet per business
    CONSTRAINT idx_business_wallets_account_number UNIQUE (virtual_account_number),
    CONSTRAINT idx_business_wallets_account_reference UNIQUE (virtual_account_reference)
);

CREATE INDEX IF NOT EXISTS idx_business_wallets_business_id_lookup ON business_wallets(business_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_business_wallets_provider ON business_wallets(provider) WHERE deleted_at IS NULL;

-- Table: wallet_transactions
-- Provides complete audit trail of all wallet activity (deposits, withdrawals, fees, refunds)
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- Business relationship
    business_id INTEGER NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,

    -- Transaction details
    transaction_type VARCHAR(20) NOT NULL, -- deposit, withdrawal, fee, refund
    amount BIGINT NOT NULL, -- Always positive (use type to indicate direction)
    balance_before BIGINT NOT NULL, -- Balance before this transaction
    balance_after BIGINT NOT NULL, -- Balance after this transaction
    currency VARCHAR(10) NOT NULL DEFAULT 'NGN',

    -- Reference tracking
    reference VARCHAR(100) UNIQUE NOT NULL, -- Our internal reference
    provider_reference VARCHAR(100), -- Provider's transaction reference (e.g., KPY-PAY-xxx)
    description VARCHAR(500), -- Human-readable description

    -- Link to related transfer (if this is a withdrawal)
    transfer_id INTEGER REFERENCES transfers(id) ON DELETE SET NULL, -- Nullable - only set for withdrawals

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'completed', -- completed, pending, failed

    -- Timestamps
    processed_at TIMESTAMP, -- When transaction was processed

    -- Indexes
    CONSTRAINT idx_wallet_transactions_reference UNIQUE (reference),
    CONSTRAINT idx_wallet_transactions_business_id ON wallet_transactions(business_id),
    CONSTRAINT idx_wallet_transactions_transfer_id ON wallet_transactions(transfer_id) WHERE transfer_id IS NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_wallet_transactions_business_id_lookup ON wallet_transactions(business_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_type ON wallet_transactions(transaction_type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_provider_reference ON wallet_transactions(provider_reference) WHERE provider_reference IS NOT NULL;
