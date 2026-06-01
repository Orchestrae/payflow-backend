ALTER TABLE payroll_runs ADD COLUMN IF NOT EXISTS processing_job_id VARCHAR(100);
ALTER TABLE payroll_runs ADD COLUMN IF NOT EXISTS processing_error VARCHAR(1000);
