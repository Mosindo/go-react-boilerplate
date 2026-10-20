CREATE TABLE IF NOT EXISTS sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  user_agent TEXT NULL,
  ip_address INET NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ NULL,
  last_used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token_hash
  ON sessions (token_hash);

CREATE INDEX IF NOT EXISTS idx_sessions_user_created_at
  ON sessions (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_sessions_user_expires_at
  ON sessions (user_id, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_sessions_active_expires_at
  ON sessions (expires_at)
  WHERE revoked_at IS NULL;
