CREATE TABLE IF NOT EXISTS conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind TEXT NOT NULL DEFAULT 'direct',
  direct_key TEXT NULL,
  created_by_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  title TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  archived_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_direct_key
  ON conversations (direct_key);

CREATE INDEX IF NOT EXISTS idx_conversations_kind_created_at
  ON conversations (kind, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_conversations_updated_at
  ON conversations (updated_at DESC, id DESC);
