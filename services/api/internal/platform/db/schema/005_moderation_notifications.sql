CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id);

CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  match_id UUID NULL REFERENCES matches(id) ON DELETE SET NULL,
  reason TEXT NOT NULL CHECK (reason IN ('spam', 'fake_profile', 'harassment', 'inappropriate_content', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (reporter_id <> reported_id),
  UNIQUE (reporter_id, reported_id)
);
CREATE INDEX idx_reports_reported_status ON reports (reported_id, status);

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL CHECK (type IN ('match', 'message')),
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  read_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC, id DESC);
-- One rolling unread "new message" notification per conversation.
CREATE UNIQUE INDEX idx_notifications_unread_message
  ON notifications (user_id, (data->>'matchId'))
  WHERE type = 'message' AND read_at IS NULL;
