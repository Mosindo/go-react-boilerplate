-- The product is a single-community dating app: users no longer belong to a
-- tenant organization. The column is kept (nullable) so applied history stays valid.
ALTER TABLE users ALTER COLUMN organization_id DROP NOT NULL;

-- Emails are always stored normalized.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_normalized;
ALTER TABLE users
  ADD CONSTRAINT users_email_normalized CHECK (email = lower(btrim(email))) NOT VALID;

-- Account recovery: short numeric code sent by email, stored hashed, limited attempts.
CREATE TABLE IF NOT EXISTS password_resets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_resets_user_created_at
  ON password_resets (user_id, created_at DESC);
