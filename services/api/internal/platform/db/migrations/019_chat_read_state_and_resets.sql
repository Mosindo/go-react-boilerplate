-- Per-participant "clear conversation" marker: messages created before it are
-- hidden for that participant only.
ALTER TABLE conversation_participants
  ADD COLUMN IF NOT EXISTS cleared_at TIMESTAMPTZ NULL;

-- Unread counters look up messages from the other participant after last_read_at.
CREATE INDEX IF NOT EXISTS idx_messages_conversation_created_at_asc
  ON messages (conversation_id, created_at, id);

CREATE TABLE IF NOT EXISTS password_resets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_resets_user_created_at
  ON password_resets (user_id, created_at DESC);

-- Notifications carry a small JSON payload (e.g. the peer user id) so the
-- client can deep link; data already exists, add a lookup for coalescing.
CREATE INDEX IF NOT EXISTS idx_notifications_user_type_unread
  ON notifications (user_id, type, created_at DESC)
  WHERE is_read = FALSE;
