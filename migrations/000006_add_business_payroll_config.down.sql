-- Remove payroll workflow configuration fields
ALTER TABLE businesses
DROP COLUMN IF EXISTS payroll_requires_approval,
DROP COLUMN IF EXISTS payroll_auto_process;
