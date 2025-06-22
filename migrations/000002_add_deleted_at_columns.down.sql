BEGIN;

-- Remove deleted_at columns
ALTER TABLE businesses DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE deduction_rules DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE cadres DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE earning_components DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE employees DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE payroll_runs DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE payroll_run_entries DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE payroll_run_entry_details DROP COLUMN IF EXISTS deleted_at;

COMMIT;
