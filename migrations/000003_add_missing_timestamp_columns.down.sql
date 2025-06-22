BEGIN;

-- Remove timestamp columns
ALTER TABLE earning_components DROP COLUMN IF EXISTS created_at;
ALTER TABLE earning_components DROP COLUMN IF EXISTS updated_at;

ALTER TABLE payroll_run_entries DROP COLUMN IF EXISTS created_at;
ALTER TABLE payroll_run_entries DROP COLUMN IF EXISTS updated_at;

ALTER TABLE payroll_run_entry_details DROP COLUMN IF EXISTS created_at;
ALTER TABLE payroll_run_entry_details DROP COLUMN IF EXISTS updated_at;

COMMIT;
