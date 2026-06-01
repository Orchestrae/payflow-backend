-- Add bank account verification fields to employees
ALTER TABLE employees ADD COLUMN IF NOT EXISTS bank_account_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE employees ADD COLUMN IF NOT EXISTS bank_account_name VARCHAR(255) DEFAULT '';
