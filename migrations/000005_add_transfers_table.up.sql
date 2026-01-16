-- Create transfers table for provider-agnostic transfer records
CREATE TABLE IF NOT EXISTS transfers (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Business relationship
    business_id INTEGER REFERENCES businesses(id),
    
    -- Core transfer details
    reference VARCHAR(100) NOT NULL UNIQUE,
    amount VARCHAR(20) NOT NULL,
    currency VARCHAR(10) DEFAULT 'NGN',
    narration VARCHAR(500),
    
    -- Recipient details
    recipient_bank_code VARCHAR(10),
    recipient_account_number VARCHAR(20),
    recipient_account_name VARCHAR(255),
    recipient_email VARCHAR(255),
    
    -- Provider and status
    provider VARCHAR(20),
    status VARCHAR(20) DEFAULT 'pending',
    transaction_id VARCHAR(100),
    provider_status VARCHAR(50),
    provider_message VARCHAR(500),
    fee VARCHAR(20),
    
    -- Processing tracking
    processed_at TIMESTAMP WITH TIME ZONE,
    processing_error VARCHAR(1000)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_transfers_business_id ON transfers(business_id);
CREATE INDEX IF NOT EXISTS idx_transfers_recipient_account ON transfers(recipient_account_number);
CREATE INDEX IF NOT EXISTS idx_transfers_status ON transfers(status);
CREATE INDEX IF NOT EXISTS idx_transfers_created_at ON transfers(created_at DESC);
