-- Remove description field
ALTER TABLE payroll_run_entry_details
DROP COLUMN IF EXISTS description;
