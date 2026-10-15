-- A conversation exists only for a match and disappears with it (unmatch, block, deletion).
ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS match_id UUID NULL REFERENCES matches(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_match
  ON conversations (match_id)
  WHERE match_id IS NOT NULL;

-- Local deletion: a participant can hide a conversation from their own list.
-- It resurfaces when a newer message arrives.
ALTER TABLE conversation_participants
  ADD COLUMN IF NOT EXISTS hidden_at TIMESTAMPTZ NULL;

ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_content_length;
ALTER TABLE messages
  ADD CONSTRAINT messages_content_length CHECK (char_length(content) BETWEEN 1 AND 2000) NOT VALID;

-- Latest-message lookups and unread counts per conversation.
CREATE INDEX IF NOT EXISTS idx_messages_conversation_created_asc
  ON messages (conversation_id, created_at, id);
