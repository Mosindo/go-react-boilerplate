-- Dating app foundation: users get a birth date (age gate) and activity tracking.
-- Multi-tenancy is no longer used by the product, so organization_id becomes optional.
ALTER TABLE users ADD COLUMN IF NOT EXISTS birth_date DATE NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE users ALTER COLUMN organization_id DROP NOT NULL;

CREATE TABLE IF NOT EXISTS password_resets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_password_resets_token_hash ON password_resets (token_hash);
CREATE INDEX IF NOT EXISTS idx_password_resets_user_created_at ON password_resets (user_id, created_at DESC);
