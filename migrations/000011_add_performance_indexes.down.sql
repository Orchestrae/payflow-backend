DROP INDEX IF EXISTS idx_cadres_business_id;
DROP INDEX IF EXISTS idx_deduction_rules_business_id;
DROP INDEX IF EXISTS idx_deduction_rules_cadre_id;
DROP INDEX IF EXISTS idx_payroll_run_entries_payroll_run_id;

ALTER TABLE business_wallets DROP CONSTRAINT IF EXISTS chk_wallet_balance_non_negative;
ALTER TABLE business_wallets DROP CONSTRAINT IF EXISTS chk_wallet_locked_non_negative;
