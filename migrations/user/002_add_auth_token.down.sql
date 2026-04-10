DROP INDEX IF EXISTS idx_users_auth_token;
ALTER TABLE users DROP COLUMN IF EXISTS auth_token;
