CREATE TABLE IF NOT EXISTS conversation_participants (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member',
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_read_at TIMESTAMPTZ NULL,
  PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_participants_user_joined_at
  ON conversation_participants (user_id, joined_at DESC, conversation_id);

CREATE INDEX IF NOT EXISTS idx_conversation_participants_conversation_joined_at
  ON conversation_participants (conversation_id, joined_at DESC, user_id);
