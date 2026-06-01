-- Add email verification fields to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_token VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_expiry TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_users_email_verification_token ON users(email_verification_token);
