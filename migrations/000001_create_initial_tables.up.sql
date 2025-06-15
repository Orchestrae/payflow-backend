-- migrations/000001_create_initial_tables.up.sql
BEGIN;

-- For UUID generation if needed later
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE businesses (
                            id SERIAL PRIMARY KEY,
                            admin_id INT NOT NULL, -- We'll add the foreign key constraint after users table is created
                            name VARCHAR(255) NOT NULL,
                            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE user_role AS ENUM ('admin', 'operator', 'approver');

CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       business_id INT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
                       email VARCHAR(255) UNIQUE NOT NULL,
                       password_hash VARCHAR(255) NOT NULL,
                       role user_role NOT NULL,
                       is_verified BOOLEAN NOT NULL DEFAULT FALSE,
                       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                       updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Now add the foreign key to businesses
ALTER TABLE businesses ADD CONSTRAINT fk_admin_user FOREIGN KEY (admin_id) REFERENCES users(id) ON DELETE RESTRICT;

CREATE TABLE deduction_rules (
                                 id SERIAL PRIMARY KEY,
                                 business_id INT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
                                 name VARCHAR(100) NOT NULL,
                                 type VARCHAR(20) NOT NULL, -- 'percentage' or 'flat'
                                 value NUMERIC(10, 2) NOT NULL,
                                 calculation_basis VARCHAR(20) NOT NULL, -- 'gross_pay' or 'basic_pay'
                                 UNIQUE(business_id, name)
);

CREATE TABLE cadres (
                        id SERIAL PRIMARY KEY,
                        business_id INT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
                        name VARCHAR(255) NOT NULL,
                        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                        UNIQUE(business_id, name)
);

CREATE TABLE cadre_deduction_rules (
                                       cadre_id INT NOT NULL REFERENCES cadres(id) ON DELETE CASCADE,
                                       deduction_rule_id INT NOT NULL REFERENCES deduction_rules(id) ON DELETE CASCADE,
                                       PRIMARY KEY (cadre_id, deduction_rule_id)
);

CREATE TABLE earning_components (
                                    id SERIAL PRIMARY KEY,
                                    cadre_id INT NOT NULL REFERENCES cadres(id) ON DELETE CASCADE,
                                    name VARCHAR(100) NOT NULL,
                                    amount BIGINT NOT NULL -- Stored in smallest currency unit (e.g., cents)
);

CREATE TABLE employees (
                           id SERIAL PRIMARY KEY,
                           business_id INT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
                           cadre_id INT NOT NULL REFERENCES cadres(id) ON DELETE RESTRICT,
                           full_name VARCHAR(255) NOT NULL,
                           email VARCHAR(255) NOT NULL,
                           bank_name VARCHAR(100) NOT NULL,
                           bank_account_number VARCHAR(50) NOT NULL,
                           is_active BOOLEAN NOT NULL DEFAULT TRUE,
                           created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                           updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                           UNIQUE(business_id, email)
);

CREATE TYPE payroll_status AS ENUM ('draft', 'pending_approval', 'approved', 'processing', 'completed', 'rejected', 'failed');

CREATE TABLE payroll_runs (
                              id SERIAL PRIMARY KEY,
                              business_id INT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
                              period DATE NOT NULL,
                              status payroll_status NOT NULL DEFAULT 'draft',
                              total_gross_pay BIGINT NOT NULL DEFAULT 0,
                              total_deductions BIGINT NOT NULL DEFAULT 0,
                              total_net_pay BIGINT NOT NULL DEFAULT 0,
                              scheduled_for DATE NOT NULL,
                              processed_at TIMESTAMPTZ,
                              payment_reference VARCHAR(255),
                              rejection_reason TEXT,
                              created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                              updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE payroll_run_entries (
                                     id SERIAL PRIMARY KEY,
                                     payroll_run_id INT NOT NULL REFERENCES payroll_runs(id) ON DELETE CASCADE,
                                     employee_id INT NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
                                     gross_pay BIGINT NOT NULL,
                                     total_deductions BIGINT NOT NULL,
                                     bonuses BIGINT NOT NULL DEFAULT 0,
                                     net_pay BIGINT NOT NULL
);

CREATE TYPE payroll_entry_detail_type AS ENUM ('earning', 'deduction', 'bonus');

CREATE TABLE payroll_run_entry_details (
                                           id SERIAL PRIMARY KEY,
                                           payroll_run_entry_id INT NOT NULL REFERENCES payroll_run_entries(id) ON DELETE CASCADE,
                                           type payroll_entry_detail_type NOT NULL,
                                           name VARCHAR(100) NOT NULL,
                                           amount BIGINT NOT NULL
);

-- Add indexes for frequently queried columns
CREATE INDEX idx_users_business_id ON users(business_id);
CREATE INDEX idx_employees_business_id ON employees(business_id);
CREATE INDEX idx_payroll_runs_business_id_status ON payroll_runs(business_id, status);

COMMIT;