CREATE TABLE conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  last_message_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_participants (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_at TIMESTAMPTZ NULL,
  -- Local deletion: messages older than cleared_at are hidden for this user only.
  cleared_at TIMESTAMPTZ NULL,
  PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, conversation_id);

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  CONSTRAINT chk_messages_body CHECK (length(btrim(body)) BETWEEN 1 AND 2000)
);

CREATE INDEX idx_messages_conversation_created ON messages (conversation_id, created_at DESC, id DESC);
