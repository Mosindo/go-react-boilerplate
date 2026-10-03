CREATE TABLE IF NOT EXISTS swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CHECK (swiper_id <> target_id)
);

CREATE INDEX IF NOT EXISTS idx_swipes_target_action
  ON swipes (target_id, action, created_at DESC);

-- A match doubles as the conversation between its two users.
-- user_a < user_b guarantees a single row per pair.
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_message_at TIMESTAMPTZ NULL,
  hidden_a_at TIMESTAMPTZ NULL,
  hidden_b_at TIMESTAMPTZ NULL,
  CHECK (user_a < user_b),
  UNIQUE (user_a, user_b)
);

CREATE INDEX IF NOT EXISTS idx_matches_user_a ON matches (user_a, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_matches_user_b ON matches (user_b, created_at DESC);

CREATE TABLE IF NOT EXISTS chat_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_match_created_at
  ON chat_messages (match_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_chat_messages_unread
  ON chat_messages (match_id, sender_id) WHERE read_at IS NULL;
