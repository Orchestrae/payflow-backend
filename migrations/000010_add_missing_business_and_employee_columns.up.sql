-- Migration: Add missing columns to businesses and employees tables
-- These columns exist in the GORM models but were never added via SQL migration

BEGIN;

-- Add missing business columns for registration flow
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS rc_number VARCHAR(50);
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS incorporation_date TIMESTAMPTZ;
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS director_bvn VARCHAR(11);
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS vfd_account_number VARCHAR(20);
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS vfd_account_name VARCHAR(255);

-- Add missing employee column for bank code (used by Korapay disbursements)
ALTER TABLE employees ADD COLUMN IF NOT EXISTS bank_code VARCHAR(10);

-- Add cadre_id to deduction_rules (domain model has it for optional cadre-specific deductions)
-- No FK constraint since deduction rules can be global (cadre_id = 0 or NULL)
ALTER TABLE deduction_rules ADD COLUMN IF NOT EXISTS cadre_id INT DEFAULT 0;

-- Add missing timestamp columns to tables that were created without them
ALTER TABLE deduction_rules ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE deduction_rules ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE deduction_rules ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE earning_components ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE earning_components ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE earning_components ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE payroll_run_entry_details ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE payroll_run_entry_details ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE payroll_run_entry_details ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

COMMIT;
