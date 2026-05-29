DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS subscription_plans;
ALTER TABLE businesses DROP COLUMN IF EXISTS subscription_tier;
ALTER TABLE businesses DROP COLUMN IF EXISTS subscription_status;
ALTER TABLE businesses DROP COLUMN IF EXISTS is_suspended;
