-- Swipes, matches, messaging extensions, safety.

CREATE TABLE IF NOT EXISTS swipes (
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (from_user_id, to_user_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_distinct CHECK (from_user_id <> to_user_id)
);
CREATE INDEX IF NOT EXISTS idx_swipes_to_user ON swipes (to_user_id, from_user_id) WHERE action = 'like';

-- user_a < user_b makes the pair canonical, so a pair can only match once.
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unmatched_at TIMESTAMPTZ NULL,
  unmatched_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_matches_order CHECK (user_a < user_b),
  CONSTRAINT uq_matches_pair UNIQUE (user_a, user_b)
);
CREATE INDEX IF NOT EXISTS idx_matches_user_a ON matches (user_a, created_at DESC) WHERE unmatched_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_matches_user_b ON matches (user_b, created_at DESC) WHERE unmatched_at IS NULL;

-- Conversations created by matching are kind = 'match' and tied to their match.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS match_id UUID NULL REFERENCES matches(id) ON DELETE CASCADE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_match ON conversations (match_id) WHERE match_id IS NOT NULL;

-- Per-user "delete conversation": messages before cleared_at are hidden for that user only.
ALTER TABLE conversation_participants ADD COLUMN IF NOT EXISTS cleared_at TIMESTAMPTZ NULL;

ALTER TABLE messages ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ NULL;
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_messages_content') THEN
    ALTER TABLE messages ADD CONSTRAINT chk_messages_content
      CHECK (char_length(btrim(content)) BETWEEN 1 AND 2000) NOT VALID;
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_messages_unread
  ON messages (conversation_id, sender_user_id) WHERE read_at IS NULL;

CREATE TABLE IF NOT EXISTS blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT chk_blocks_distinct CHECK (blocker_id <> blocked_id)
);
CREATE INDEX IF NOT EXISTS idx_blocks_blocked ON blocks (blocked_id, blocker_id);

CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  resolved_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_reports_reason CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT chk_reports_details CHECK (char_length(details) <= 1000)
);
CREATE INDEX IF NOT EXISTS idx_reports_status_created ON reports (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_reported ON reports (reported_user_id);
