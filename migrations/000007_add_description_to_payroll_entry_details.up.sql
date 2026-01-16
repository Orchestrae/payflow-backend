-- Add description field to payroll_run_entry_details for historical tracking
ALTER TABLE payroll_run_entry_details
ADD COLUMN description VARCHAR(500) DEFAULT '';

-- Add comment for clarity
COMMENT ON COLUMN payroll_run_entry_details.description IS 'Optional description for adjustment items, useful for historical tracking';
