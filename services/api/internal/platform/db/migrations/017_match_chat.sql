-- Chat is restricted to matches: one conversation per match.
ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS match_id UUID NULL REFERENCES matches(id) ON DELETE CASCADE;
ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_match_id ON conversations (match_id);

-- Local deletion: a participant can hide a conversation; a new message un-hides it.
ALTER TABLE conversation_participants
  ADD COLUMN IF NOT EXISTS hidden_at TIMESTAMPTZ NULL;

ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'messages_content_length') THEN
    ALTER TABLE messages
      ADD CONSTRAINT messages_content_length CHECK (char_length(content) BETWEEN 1 AND 2000) NOT VALID;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_messages_unread
  ON messages (conversation_id, recipient_user_id)
  WHERE read_at IS NULL;
