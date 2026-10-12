-- One row per (swiper, target): a like or a pass. The primary key prevents duplicates.
CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CHECK (swiper_id <> target_id)
);
CREATE INDEX idx_swipes_target_likes ON swipes (target_id, swiper_id) WHERE action = 'like';

-- A match is also the conversation: user_a < user_b makes the pair canonical.
CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_message_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (user_a < user_b),
  UNIQUE (user_a, user_b)
);
CREATE INDEX idx_matches_user_b ON matches (user_b);

-- Per-participant conversation state (read marker, local history clearing).
CREATE TABLE match_members (
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  cleared_at TIMESTAMPTZ NULL,
  PRIMARY KEY (match_id, user_id)
);
CREATE INDEX idx_match_members_user ON match_members (user_id, match_id);

CREATE TABLE chat_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_chat_messages_match_created ON chat_messages (match_id, created_at DESC, id DESC);
