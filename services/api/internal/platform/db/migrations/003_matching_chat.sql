CREATE TABLE swipes (
  swiper_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action     TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT swipes_not_self CHECK (swiper_id <> target_id)
);
CREATE INDEX idx_swipes_target_action ON swipes (target_id, action, created_at DESC);

CREATE TABLE matches (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT matches_ordered CHECK (user_a < user_b),
  CONSTRAINT matches_unique UNIQUE (user_a, user_b)
);
CREATE INDEX idx_matches_user_b ON matches (user_b, created_at DESC);
CREATE INDEX idx_matches_user_a ON matches (user_a, created_at DESC);

CREATE TABLE conversations (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id        UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_participants (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  hidden_at       TIMESTAMPTZ NULL,
  PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, conversation_id);

CREATE TABLE messages (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body            TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT messages_body_len CHECK (char_length(body) BETWEEN 1 AND 2000)
);
CREATE INDEX idx_messages_conversation_created ON messages (conversation_id, created_at DESC, id DESC);

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT blocks_not_self CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id);

CREATE TABLE reports (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason      TEXT NOT NULL,
  details     TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT 'open',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT reports_reason CHECK (reason IN ('spam', 'fake', 'harassment', 'inappropriate', 'underage', 'other')),
  CONSTRAINT reports_status CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT reports_not_self CHECK (reporter_id <> reported_id),
  CONSTRAINT reports_details_len CHECK (char_length(details) <= 1000)
);
CREATE INDEX idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE INDEX idx_reports_status ON reports (status, created_at DESC);

CREATE TABLE notifications (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type       TEXT NOT NULL,
  actor_id   UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  ref_id     UUID NULL,
  read_at    TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT notifications_type CHECK (type IN ('match', 'message'))
);
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_unread ON notifications (user_id) WHERE read_at IS NULL;
