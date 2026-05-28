-- User invitation and password reset tokens
ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token_expiry TIMESTAMP;
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_token VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_accepted BOOLEAN DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_users_reset_token ON users(reset_token) WHERE reset_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_invite_token ON users(invite_token) WHERE invite_token IS NOT NULL;
