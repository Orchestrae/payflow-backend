-- Earning component classification for pension base calculation
ALTER TABLE earning_components ADD COLUMN IF NOT EXISTS component_type VARCHAR(20) DEFAULT 'other';

-- Employee statutory identifiers
ALTER TABLE employees ADD COLUMN IF NOT EXISTS tin VARCHAR(20);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS pension_rsa_pin VARCHAR(30);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS nhf_number VARCHAR(30);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS annual_rent_paid BIGINT DEFAULT 0;

-- Business statutory configuration
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS pension_enabled BOOLEAN DEFAULT false;
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS nhf_enabled BOOLEAN DEFAULT false;
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS nsitf_enabled BOOLEAN DEFAULT false;
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS paye_enabled BOOLEAN DEFAULT true;

-- Payroll entry employer costs
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS employer_pension BIGINT DEFAULT 0;
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS employer_nsitf BIGINT DEFAULT 0;
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS total_employer_cost BIGINT DEFAULT 0;
ALTER TABLE payroll_run_entries ADD COLUMN IF NOT EXISTS total_cost_to_company BIGINT DEFAULT 0;

-- Payroll run aggregate employer costs
ALTER TABLE payroll_runs ADD COLUMN IF NOT EXISTS total_employer_costs BIGINT DEFAULT 0;
ALTER TABLE payroll_runs ADD COLUMN IF NOT EXISTS total_cost_to_company BIGINT DEFAULT 0;

-- Extend payroll detail type enum for statutory deductions and employer costs
ALTER TYPE payroll_entry_detail_type ADD VALUE IF NOT EXISTS 'statutory_deduction';
ALTER TYPE payroll_entry_detail_type ADD VALUE IF NOT EXISTS 'employer_cost';
