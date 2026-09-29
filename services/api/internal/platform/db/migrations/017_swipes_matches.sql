CREATE TABLE IF NOT EXISTS swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_distinct CHECK (swiper_id <> target_id)
);

-- Reverse lookup: "who liked me" and reciprocal-like detection.
CREATE INDEX IF NOT EXISTS idx_swipes_target_action ON swipes (target_id, action, created_at DESC);

-- A match is an unordered pair, stored ordered (user_a < user_b) so it is unique by construction.
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_matches_order CHECK (user_a < user_b),
  CONSTRAINT uq_matches_pair UNIQUE (user_a, user_b)
);

CREATE INDEX IF NOT EXISTS idx_matches_user_b ON matches (user_b, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_matches_user_a ON matches (user_a, created_at DESC);
