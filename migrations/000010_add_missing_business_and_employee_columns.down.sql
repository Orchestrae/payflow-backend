-- Rollback: Remove columns added in migration 000010

BEGIN;

ALTER TABLE businesses DROP COLUMN IF EXISTS rc_number;
ALTER TABLE businesses DROP COLUMN IF EXISTS incorporation_date;
ALTER TABLE businesses DROP COLUMN IF EXISTS director_bvn;
ALTER TABLE businesses DROP COLUMN IF EXISTS vfd_account_number;
ALTER TABLE businesses DROP COLUMN IF EXISTS vfd_account_name;

ALTER TABLE employees DROP COLUMN IF EXISTS bank_code;

ALTER TABLE deduction_rules DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE earning_components DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE payroll_run_entries DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE payroll_run_entry_details DROP COLUMN IF EXISTS deleted_at;

COMMIT;
