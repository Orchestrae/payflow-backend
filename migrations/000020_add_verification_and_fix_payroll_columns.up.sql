-- Business verification fields
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS director_bvn_last4 VARCHAR(4);
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS bvn_verified BOOLEAN DEFAULT false;
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS rc_verified BOOLEAN DEFAULT false;
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT false;

-- Fix missing payroll run employer cost columns
ALTER TABLE payroll_runs ADD COLUMN IF NOT EXISTS total_employer_costs BIGINT DEFAULT 0;
ALTER TABLE payroll_runs ADD COLUMN IF NOT EXISTS total_cost_to_company BIGINT DEFAULT 0;

-- Fix missing payroll run entry employer cost columns
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS employer_pension BIGINT DEFAULT 0;
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS employer_nsitf BIGINT DEFAULT 0;
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS total_employer_cost BIGINT DEFAULT 0;
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS total_cost_to_company BIGINT DEFAULT 0;
