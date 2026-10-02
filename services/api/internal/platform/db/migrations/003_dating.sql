CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CHECK (swiper_id <> target_id)
);
CREATE INDEX idx_swipes_target_likes ON swipes (target_id, swiper_id) WHERE action = 'like';

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  a_cleared_at TIMESTAMPTZ NULL,
  b_cleared_at TIMESTAMPTZ NULL,
  last_message_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (user_a < user_b),
  UNIQUE (user_a, user_b)
);
CREATE INDEX idx_matches_user_a ON matches (user_a, COALESCE(last_message_at, created_at) DESC);
CREATE INDEX idx_matches_user_b ON matches (user_b, COALESCE(last_message_at, created_at) DESC);

CREATE TABLE messages (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_messages_match_created ON messages (match_id, created_at DESC, id DESC);
CREATE INDEX idx_messages_unread ON messages (match_id, sender_id) WHERE read_at IS NULL;

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
  reason TEXT NOT NULL CHECK (reason IN ('spam', 'fake', 'inappropriate', 'harassment', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (reporter_id <> reported_id)
);
CREATE INDEX idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE INDEX idx_reports_status ON reports (status, created_at DESC);

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL CHECK (type IN ('match', 'message')),
  actor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);
CREATE UNIQUE INDEX idx_notifications_unread_message ON notifications (user_id, match_id)
  WHERE type = 'message' AND read_at IS NULL;
