BEGIN;

-- Add deleted_at columns for GORM soft deletes
ALTER TABLE businesses ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE deduction_rules ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE cadres ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE earning_components ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE employees ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE payroll_runs ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE payroll_run_entries ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE payroll_run_entry_details ADD COLUMN deleted_at TIMESTAMPTZ;

COMMIT;
