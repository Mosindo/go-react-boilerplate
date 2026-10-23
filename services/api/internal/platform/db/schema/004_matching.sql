-- Swipes, matches (a match is its own conversation), messages.

CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CHECK (swiper_id <> target_id)
);
CREATE INDEX idx_swipes_target_action ON swipes (target_id, action);

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unmatched_at TIMESTAMPTZ NULL,
  a_hidden_at TIMESTAMPTZ NULL,
  b_hidden_at TIMESTAMPTZ NULL,
  -- Canonical ordering guarantees one row per pair.
  CHECK (user_a < user_b),
  UNIQUE (user_a, user_b)
);
CREATE INDEX idx_matches_a_activity ON matches (user_a, last_activity_at DESC) WHERE unmatched_at IS NULL;
CREATE INDEX idx_matches_b_activity ON matches (user_b, last_activity_at DESC) WHERE unmatched_at IS NULL;

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content TEXT NOT NULL CHECK (char_length(content) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_messages_match_created ON messages (match_id, created_at DESC, id DESC);
CREATE INDEX idx_messages_unread ON messages (match_id, sender_id) WHERE read_at IS NULL;
