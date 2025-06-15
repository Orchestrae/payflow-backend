-- migrations/000001_create_initial_tables.down.sql
BEGIN;

DROP TABLE IF EXISTS payroll_run_entry_details;
DROP TABLE IF EXISTS payroll_run_entries;
DROP TABLE IF EXISTS payroll_runs;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS earning_components;
DROP TABLE IF EXISTS cadre_deduction_rules;
DROP TABLE IF EXISTS cadres;
DROP TABLE IF EXISTS deduction_rules;
ALTER TABLE businesses DROP CONSTRAINT IF EXISTS fk_admin_user;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS businesses;

DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS payroll_status;
DROP TYPE IF EXISTS payroll_entry_detail_type;

COMMIT;