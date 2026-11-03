CREATE TABLE swipes (
  actor_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action     TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (actor_id, target_id),
  CHECK (actor_id <> target_id)
);
CREATE INDEX idx_swipes_target_action ON swipes (target_id, action, created_at DESC);

CREATE TABLE matches (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (user_a < user_b),
  UNIQUE (user_a, user_b)
);
CREATE INDEX idx_matches_user_b ON matches (user_b, created_at DESC);
CREATE INDEX idx_matches_user_a ON matches (user_a, created_at DESC);

CREATE TABLE conversations (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id   UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_participants (
  conversation_id      UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_message_id BIGINT NOT NULL DEFAULT 0,
  cleared_up_to_id     BIGINT NOT NULL DEFAULT 0,
  hidden               BOOLEAN NOT NULL DEFAULT FALSE,
  PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, conversation_id);

CREATE TABLE messages (
  id              BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body            TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_messages_conversation_created ON messages (conversation_id, created_at DESC, id DESC);

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id);

CREATE TABLE reports (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason      TEXT NOT NULL CHECK (reason IN ('spam', 'fake', 'harassment', 'inappropriate', 'underage', 'other')),
  details     TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (reporter_id <> reported_id)
);
CREATE INDEX idx_reports_status_created ON reports (status, created_at DESC);
CREATE INDEX idx_reports_reported ON reports (reported_id);

CREATE TABLE notifications (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type       TEXT NOT NULL CHECK (type IN ('match', 'system')),
  actor_id   UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  match_id   UUID NULL REFERENCES matches(id) ON DELETE CASCADE,
  read_at    TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications (user_id) WHERE read_at IS NULL;
