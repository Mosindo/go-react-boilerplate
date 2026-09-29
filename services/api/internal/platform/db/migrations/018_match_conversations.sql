-- Conversations are reused for chat; a dating conversation belongs to exactly one match.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS match_id UUID NULL REFERENCES matches(id) ON DELETE CASCADE;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_match_id ON conversations (match_id) WHERE match_id IS NOT NULL;

-- hidden_at implements local ("for me only") conversation deletion.
ALTER TABLE conversation_participants ADD COLUMN IF NOT EXISTS hidden_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_messages_conversation_unread
  ON messages (conversation_id, sender_user_id, created_at DESC);
