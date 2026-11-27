-- Keep the previous refresh token hash of each session so that presenting an
-- already-rotated token (a sign of theft) can be detected and the session
-- revoked.
ALTER TABLE sessions
  ADD COLUMN IF NOT EXISTS previous_token_hash TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_previous_token_hash
  ON sessions (previous_token_hash)
  WHERE previous_token_hash IS NOT NULL;
