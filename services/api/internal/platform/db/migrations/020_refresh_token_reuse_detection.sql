-- Remember the previous refresh token of each session so that replaying an
-- already-rotated token is detected and revokes the whole session.
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS previous_token_hash TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_previous_token_hash
  ON sessions (previous_token_hash)
  WHERE previous_token_hash IS NOT NULL;
