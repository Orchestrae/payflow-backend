-- Employee self-service link + pro-rata
ALTER TABLE employees ADD COLUMN IF NOT EXISTS user_id BIGINT;
ALTER TABLE employees ADD COLUMN IF NOT EXISTS start_date TIMESTAMP;
ALTER TABLE employees ADD COLUMN IF NOT EXISTS phone_number VARCHAR(20) DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_employees_user_id ON employees(user_id) WHERE user_id IS NOT NULL;

-- Add employee role to user_role enum
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'employee';

-- Employee loans
CREATE TABLE IF NOT EXISTS employee_loans (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    business_id BIGINT NOT NULL,
    employee_id BIGINT NOT NULL,
    loan_amount BIGINT NOT NULL,
    monthly_deduction BIGINT NOT NULL,
    total_repaid BIGINT DEFAULT 0,
    remaining_balance BIGINT NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    start_date TIMESTAMP NOT NULL,
    description VARCHAR(500)
);

CREATE INDEX IF NOT EXISTS idx_employee_loans_business ON employee_loans(business_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_employee_loans_employee ON employee_loans(employee_id, status) WHERE deleted_at IS NULL;
