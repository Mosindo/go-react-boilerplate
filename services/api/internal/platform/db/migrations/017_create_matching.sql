CREATE TABLE IF NOT EXISTS swipes (
  swiper_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_user_id, target_user_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_user_id <> target_user_id)
);

-- Reverse lookup used to detect a mutual like.
CREATE INDEX IF NOT EXISTS idx_swipes_target_action ON swipes (target_user_id, action, swiper_user_id);

CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- user_a_id < user_b_id guarantees a single row per pair.
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unmatched_at TIMESTAMPTZ NULL,
  unmatched_by_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_matches_order CHECK (user_a_id < user_b_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_user_a_active ON matches (user_a_id, created_at DESC) WHERE unmatched_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_matches_user_b_active ON matches (user_b_id, created_at DESC) WHERE unmatched_at IS NULL;
