CREATE TABLE IF NOT EXISTS swipes (
  actor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (actor_id, target_id),
  CHECK (actor_id <> target_id)
);

CREATE INDEX IF NOT EXISTS idx_swipes_target_action ON swipes (target_id, action);

-- One row per pair; user_low < user_high guarantees a single canonical ordering.
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_low UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_high UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unmatched_at TIMESTAMPTZ NULL,
  unmatched_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  CHECK (user_low < user_high),
  UNIQUE (user_low, user_high)
);

CREATE INDEX IF NOT EXISTS idx_matches_user_high ON matches (user_high);

ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS match_id UUID NULL REFERENCES matches(id) ON DELETE CASCADE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_match_id ON conversations (match_id);

-- hidden_at implements local conversation deletion (per participant).
ALTER TABLE conversation_participants
  ADD COLUMN IF NOT EXISTS hidden_at TIMESTAMPTZ NULL;

ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ NULL;
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_content_length;
ALTER TABLE messages
  ADD CONSTRAINT messages_content_length CHECK (char_length(content) BETWEEN 1 AND 2000) NOT VALID;

CREATE INDEX IF NOT EXISTS idx_messages_unread
  ON messages (conversation_id, sender_user_id)
  WHERE read_at IS NULL;
