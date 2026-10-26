CREATE TABLE swipes (
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action       TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (from_user_id, to_user_id),
  CONSTRAINT swipes_not_self CHECK (from_user_id <> to_user_id)
);
CREATE INDEX idx_swipes_to_user_like ON swipes (to_user_id, created_at DESC) WHERE action = 'like';

-- A match is an unordered pair stored with user_a < user_b. The match id doubles
-- as the conversation id (one conversation per match).
CREATE TABLE matches (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at   TIMESTAMPTZ NULL,
  CONSTRAINT matches_ordered CHECK (user_a < user_b),
  CONSTRAINT matches_pair UNIQUE (user_a, user_b)
);
CREATE INDEX idx_matches_user_b ON matches (user_b, created_at DESC);
CREATE INDEX idx_matches_user_a ON matches (user_a, created_at DESC);

CREATE TABLE messages (
  id         BIGSERIAL PRIMARY KEY,
  match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body       TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at    TIMESTAMPTZ NULL
);
CREATE INDEX idx_messages_match_created ON messages (match_id, created_at DESC, id DESC);
CREATE INDEX idx_messages_unread ON messages (match_id, sender_id) WHERE read_at IS NULL;

-- Per-user conversation state: "delete conversation" only clears history for
-- the requesting user (messages created before cleared_at are hidden for them).
CREATE TABLE conversation_states (
  match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  cleared_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (match_id, user_id)
);
