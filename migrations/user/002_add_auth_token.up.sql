ALTER TABLE users ADD COLUMN auth_token TEXT NOT NULL DEFAULT '';

-- Generate base64-encoded auth tokens for existing users
-- encode(gen_random_bytes(32), 'base64') produces a 44-char base64 string from 32 random bytes
UPDATE users SET auth_token = encode(gen_random_bytes(32), 'base64') WHERE auth_token = '';

-- Ensure all future rows have a unique token
CREATE UNIQUE INDEX idx_users_auth_token ON users(auth_token) WHERE auth_token != '';
