-- Swipes, matches and chat state.

CREATE TABLE IF NOT EXISTS swipes (
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (from_user_id, to_user_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (from_user_id <> to_user_id)
);

-- "Who liked me" and mutual-like lookups.
CREATE INDEX IF NOT EXISTS idx_swipes_received_likes
  ON swipes (to_user_id, created_at DESC)
  WHERE action = 'like';

-- user_a < user_b guarantees one row per pair whatever the like order.
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_matches_order CHECK (user_a < user_b),
  CONSTRAINT uq_matches_pair UNIQUE (user_a, user_b),
  CONSTRAINT uq_matches_conversation UNIQUE (conversation_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_user_a ON matches (user_a, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_matches_user_b ON matches (user_b, created_at DESC);

ALTER TABLE conversation_participants
  ADD COLUMN IF NOT EXISTS hidden_at TIMESTAMPTZ NULL;

ALTER TABLE messages DROP CONSTRAINT IF EXISTS chk_messages_content;
ALTER TABLE messages
  ADD CONSTRAINT chk_messages_content CHECK (char_length(content) BETWEEN 1 AND 2000) NOT VALID;
