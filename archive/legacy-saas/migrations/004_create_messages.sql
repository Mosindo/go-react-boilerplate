CREATE TABLE IF NOT EXISTS messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NULL,
  sender_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  recipient_user_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS conversation_id UUID NULL;

ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_messages_pair_created_at
  ON messages (sender_user_id, recipient_user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_messages_sender_created_at
  ON messages (sender_user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_messages_recipient_created_at
  ON messages (recipient_user_id, created_at DESC, id DESC);
