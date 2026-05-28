DROP INDEX IF EXISTS idx_users_reset_token;
DROP INDEX IF EXISTS idx_users_invite_token;
ALTER TABLE users DROP COLUMN IF EXISTS reset_token;
ALTER TABLE users DROP COLUMN IF EXISTS reset_token_expiry;
ALTER TABLE users DROP COLUMN IF EXISTS invite_token;
ALTER TABLE users DROP COLUMN IF EXISTS invite_accepted;
