-- Add payroll workflow configuration fields to businesses table
ALTER TABLE businesses
ADD COLUMN payroll_requires_approval BOOLEAN DEFAULT true,
ADD COLUMN payroll_auto_process BOOLEAN DEFAULT false;

-- Add comments for clarity
COMMENT ON COLUMN businesses.payroll_requires_approval IS 'If false, payroll auto-approves after submission';
COMMENT ON COLUMN businesses.payroll_auto_process IS 'If true, approved payroll processes immediately (for testing)';
