-- Drop the case-sensitive unique constraint and replace with a
-- case-insensitive unique index so that 'joshua' and 'JOSHUA' collide.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;
CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_idx ON users (LOWER(username));
