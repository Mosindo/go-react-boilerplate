-- Expo push tokens, one row per device installation.
CREATE TABLE IF NOT EXISTS push_tokens (
  token TEXT PRIMARY KEY CHECK (char_length(token) <= 200),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  platform TEXT NOT NULL CHECK (platform IN ('ios', 'android')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_push_tokens_user ON push_tokens (user_id);
