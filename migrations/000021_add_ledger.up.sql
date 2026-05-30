-- Double-entry accounting ledger
CREATE TABLE IF NOT EXISTS ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    business_id BIGINT NOT NULL,
    transaction_id VARCHAR(100) NOT NULL,
    account_type VARCHAR(20) NOT NULL,
    entry_type VARCHAR(10) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) DEFAULT 'NGN',
    description VARCHAR(500),
    reference VARCHAR(100),
    balance_after BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT chk_ledger_amount_positive CHECK (amount > 0),
    CONSTRAINT chk_ledger_entry_type CHECK (entry_type IN ('debit', 'credit'))
);

CREATE INDEX IF NOT EXISTS idx_ledger_business ON ledger_entries(business_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_ledger_transaction ON ledger_entries(transaction_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_ledger_reference ON ledger_entries(reference) WHERE reference IS NOT NULL AND deleted_at IS NULL;
