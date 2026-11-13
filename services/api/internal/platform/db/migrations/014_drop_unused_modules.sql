-- The dating product does not ship the generic posts/comments/votes, billing
-- or metadata-only files modules. Drop their tables on databases created by
-- earlier versions of the boilerplate.
DROP TABLE IF EXISTS votes CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS files CASCADE;

-- Legacy optional columns superseded by the profiles table.
ALTER TABLE users DROP COLUMN IF EXISTS display_name;
ALTER TABLE users DROP COLUMN IF EXISTS bio;

-- Demo accounts created by cmd/seed are flagged so they can never be
-- confused with real members and can be purged in one statement.
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_users_is_demo ON users (is_demo) WHERE is_demo;
