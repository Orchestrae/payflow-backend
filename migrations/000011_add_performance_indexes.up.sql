-- Performance indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_cadres_business_id ON cadres(business_id);
CREATE INDEX IF NOT EXISTS idx_deduction_rules_business_id ON deduction_rules(business_id);
CREATE INDEX IF NOT EXISTS idx_deduction_rules_cadre_id ON deduction_rules(cadre_id);
CREATE INDEX IF NOT EXISTS idx_payroll_run_entries_payroll_run_id ON payroll_run_entries(payroll_run_id);

-- Prevent negative wallet balance at DB level (defense in depth for atomic operations)
ALTER TABLE business_wallets ADD CONSTRAINT chk_wallet_balance_non_negative CHECK (balance >= 0);
ALTER TABLE business_wallets ADD CONSTRAINT chk_wallet_locked_non_negative CHECK (locked_balance >= 0);
