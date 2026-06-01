DROP INDEX IF EXISTS idx_users_email_verification_token;
ALTER TABLE users DROP COLUMN IF EXISTS email_verification_token;
ALTER TABLE users DROP COLUMN IF EXISTS email_verification_expiry;
