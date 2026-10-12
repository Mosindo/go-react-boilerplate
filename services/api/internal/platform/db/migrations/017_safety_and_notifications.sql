CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id, blocker_id);

CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL CHECK (reason IN ('spam', 'fake', 'inappropriate', 'harassment', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  context JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (reporter_id <> reported_id)
);
CREATE INDEX idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE INDEX idx_reports_status ON reports (status, created_at DESC);

CREATE TABLE user_notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('match', 'message')),
  actor_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  match_id UUID NULL REFERENCES matches(id) ON DELETE CASCADE,
  read_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_user_notifications_user_created ON user_notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_user_notifications_unread ON user_notifications (user_id) WHERE read_at IS NULL;
-- At most one unread "message" notification per conversation, to avoid notification spam.
CREATE UNIQUE INDEX idx_user_notifications_unread_message ON user_notifications (user_id, match_id)
  WHERE kind = 'message' AND read_at IS NULL;
