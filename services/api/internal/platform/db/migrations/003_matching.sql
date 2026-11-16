CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_id <> target_id)
);

-- Reverse lookup: "did the target already like me?" and "who liked me".
CREATE INDEX idx_swipes_target_action ON swipes (target_id, action, swiper_id);

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Canonical ordering guarantees a single row per pair.
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_message_at TIMESTAMPTZ NULL,
  unmatched_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_matches_order CHECK (user_a_id < user_b_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX idx_matches_user_a_activity ON matches (user_a_id, (COALESCE(last_message_at, created_at)) DESC) WHERE unmatched_at IS NULL;
CREATE INDEX idx_matches_user_b_activity ON matches (user_b_id, (COALESCE(last_message_at, created_at)) DESC) WHERE unmatched_at IS NULL;

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (length(body) BETWEEN 1 AND 2000),
  -- Client-generated id making retries idempotent.
  client_message_id UUID NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_messages_match_created_at ON messages (match_id, created_at DESC, id DESC);
CREATE INDEX idx_messages_unread ON messages (match_id, sender_id) WHERE read_at IS NULL;
CREATE UNIQUE INDEX idx_messages_client_id ON messages (sender_id, client_message_id) WHERE client_message_id IS NOT NULL;
