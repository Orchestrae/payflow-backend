BEGIN;

-- Add missing timestamp columns for tables that don't use gorm.Model
-- but need created_at and updated_at for consistency

-- EarningComponent needs created_at and updated_at (it doesn't use gorm.Model in domain)
ALTER TABLE earning_components ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE earning_components ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- PayrollRunEntry needs created_at and updated_at (it doesn't use gorm.Model in domain)
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- PayrollRunEntryDetail needs created_at and updated_at (it doesn't use gorm.Model in domain)
ALTER TABLE payroll_run_entry_details ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE payroll_run_entry_details ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

COMMIT;
