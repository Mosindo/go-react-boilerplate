-- Dating pivot: account status/activity on users; legacy SaaS structures (organizations,
-- billing, posts, comments, votes) are dropped. All statements are no-ops on a fresh database.
DROP TABLE IF EXISTS subscriptions, comments, votes, posts CASCADE;
ALTER TABLE users DROP COLUMN IF EXISTS organization_id;
DROP TABLE IF EXISTS organizations CASCADE;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active',
  ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE users
  ADD CONSTRAINT users_status_check CHECK (status IN ('active', 'suspended'));

ALTER TABLE users
  DROP COLUMN IF EXISTS display_name,
  DROP COLUMN IF EXISTS bio;
