-- Add employee_id to users table for employee self-service login
ALTER TABLE users ADD COLUMN IF NOT EXISTS employee_id BIGINT REFERENCES employees(id);
CREATE INDEX IF NOT EXISTS idx_users_employee_id ON users(employee_id);
