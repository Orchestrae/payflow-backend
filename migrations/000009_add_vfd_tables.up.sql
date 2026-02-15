-- Migration: Add VFD webhook notifications and transfer records tables
-- Created: 2026-01-14
-- These tables were previously only created via auto-migration
-- This migration ensures they exist when auto-migration is disabled

-- Table: vfd_webhook_notifications
-- Stores VFD webhook notifications (actively used)
CREATE TABLE IF NOT EXISTS vfd_webhook_notifications (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Business relationship
    business_id INTEGER,

    -- Webhook notification details
    reference VARCHAR(255),
    amount VARCHAR(50),
    account_number VARCHAR(20),
    originator_account_number VARCHAR(20),
    originator_account_name VARCHAR(255),
    originator_bank VARCHAR(10),
    originator_narration VARCHAR(500),
    timestamp TIMESTAMP WITH TIME ZONE, -- Note: stored as single timestamp (GORM Model.CreatedAt)
    transaction_channel VARCHAR(10),
    session_id VARCHAR(50),
    initial_credit_request BOOLEAN DEFAULT FALSE,
    status VARCHAR(20) DEFAULT 'pending',
    processed_at TIMESTAMP WITH TIME ZONE, -- Note: stored as single timestamp (GORM Model.CreatedAt)
    processing_error VARCHAR(1000),

    -- Indexes
    CONSTRAINT idx_vfd_webhook_notifications_reference UNIQUE (reference)
);

CREATE INDEX IF NOT EXISTS idx_vfd_webhook_notifications_business_id ON vfd_webhook_notifications(business_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_vfd_webhook_notifications_account_number ON vfd_webhook_notifications(account_number) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_vfd_webhook_notifications_session_id ON vfd_webhook_notifications(session_id) WHERE deleted_at IS NULL;

-- Table: transfer_records (VFD-specific, deprecated but kept for backward compatibility)
-- Stores VFD-specific transfer records
-- Note: This repository is deprecated - use 'transfers' table instead
CREATE TABLE IF NOT EXISTS transfer_records (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Business relationship
    business_id INTEGER,

    -- From account details
    from_account VARCHAR(20),
    from_client_id VARCHAR(20),
    from_client VARCHAR(255),
    from_savings_id VARCHAR(20),
    from_bvn VARCHAR(11),

    -- To account details
    to_client_id VARCHAR(20),
    to_client VARCHAR(255),
    to_savings_id VARCHAR(20),
    to_session VARCHAR(50),
    to_bvn VARCHAR(11),
    to_account VARCHAR(20),
    to_bank VARCHAR(10),

    -- Transfer details
    amount VARCHAR(20),
    remark VARCHAR(500),
    transfer_type VARCHAR(10),
    reference VARCHAR(100) UNIQUE,
    txn_id VARCHAR(100),
    session_id VARCHAR(50),

    -- Status
    status VARCHAR(20) DEFAULT 'pending',
    vfd_status VARCHAR(10),
    vfd_message VARCHAR(255),
    processed_at TIMESTAMP WITH TIME ZONE, -- Note: stored as single timestamp (GORM Model.CreatedAt)
    processing_error VARCHAR(1000),

    -- Indexes
    CONSTRAINT idx_transfer_records_reference UNIQUE (reference)
);

CREATE INDEX IF NOT EXISTS idx_transfer_records_business_id ON transfer_records(business_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_transfer_records_from_account ON transfer_records(from_account) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_transfer_records_to_account ON transfer_records(to_account) WHERE deleted_at IS NULL;
