CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_id <> target_id)
);

-- Reverse lookup used to detect reciprocal likes and to boost people who liked you.
CREATE INDEX idx_swipes_target_likes ON swipes (target_id, swiper_id) WHERE action = 'like';

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  -- Canonical ordering guarantees a single row per pair.
  CONSTRAINT chk_matches_ordered CHECK (user_a < user_b),
  CONSTRAINT uq_matches_pair UNIQUE (user_a, user_b)
);

CREATE INDEX idx_matches_user_b ON matches (user_b, created_at DESC);
CREATE INDEX idx_matches_user_a_created ON matches (user_a, created_at DESC);
