ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS conversation_id UUID NULL;

ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE messages
SET updated_at = created_at
WHERE updated_at IS NULL;

INSERT INTO conversations (kind, direct_key, created_by_user_id, created_at, updated_at)
SELECT
  'direct',
  pair_key,
  NULL,
  MIN(created_at),
  MAX(created_at)
FROM (
  SELECT
    CASE
      WHEN sender_user_id::text < recipient_user_id::text
        THEN sender_user_id::text || ':' || recipient_user_id::text
      ELSE recipient_user_id::text || ':' || sender_user_id::text
    END AS pair_key,
    created_at
  FROM messages
  WHERE recipient_user_id IS NOT NULL
) pairs
GROUP BY pair_key
ON CONFLICT (direct_key) DO NOTHING;

INSERT INTO conversation_participants (conversation_id, user_id, joined_at)
SELECT
  c.id,
  pair_members.user_id,
  MIN(pair_members.created_at)
FROM (
  SELECT
    CASE
      WHEN sender_user_id::text < recipient_user_id::text
        THEN sender_user_id::text || ':' || recipient_user_id::text
      ELSE recipient_user_id::text || ':' || sender_user_id::text
    END AS pair_key,
    sender_user_id AS user_id,
    created_at
  FROM messages
  WHERE recipient_user_id IS NOT NULL

  UNION ALL

  SELECT
    CASE
      WHEN sender_user_id::text < recipient_user_id::text
        THEN sender_user_id::text || ':' || recipient_user_id::text
      ELSE recipient_user_id::text || ':' || sender_user_id::text
    END AS pair_key,
    recipient_user_id AS user_id,
    created_at
  FROM messages
  WHERE recipient_user_id IS NOT NULL
) pair_members
JOIN conversations c
  ON c.direct_key = pair_members.pair_key
GROUP BY c.id, pair_members.user_id
ON CONFLICT (conversation_id, user_id) DO NOTHING;

UPDATE messages m
SET conversation_id = c.id
FROM conversations c
WHERE m.conversation_id IS NULL
  AND m.recipient_user_id IS NOT NULL
  AND c.direct_key = CASE
    WHEN m.sender_user_id::text < m.recipient_user_id::text
      THEN m.sender_user_id::text || ':' || m.recipient_user_id::text
    ELSE m.recipient_user_id::text || ':' || m.sender_user_id::text
  END;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_messages_conversation_id'
  ) THEN
    ALTER TABLE messages
      ADD CONSTRAINT fk_messages_conversation_id
      FOREIGN KEY (conversation_id)
      REFERENCES conversations(id)
      ON DELETE CASCADE;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_messages_conversation_created_at
  ON messages (conversation_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_sender_created_at
  ON messages (conversation_id, sender_user_id, created_at DESC, id DESC);
