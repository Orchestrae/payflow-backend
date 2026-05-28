DROP TABLE IF EXISTS employee_loans;
ALTER TABLE employees DROP COLUMN IF EXISTS user_id;
ALTER TABLE employees DROP COLUMN IF EXISTS start_date;
ALTER TABLE employees DROP COLUMN IF EXISTS phone_number;
