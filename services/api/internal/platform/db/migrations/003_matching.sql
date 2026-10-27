CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT swipes_action_valid CHECK (action IN ('like', 'pass')),
  CONSTRAINT swipes_not_self CHECK (swiper_id <> target_id)
);

CREATE INDEX idx_swipes_incoming_likes ON swipes (target_id, swiper_id) WHERE action = 'like';

-- user_a < user_b guarantees a single row per pair and no inconsistent duplicates.
CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT matches_ordered CHECK (user_a < user_b),
  CONSTRAINT matches_pair_unique UNIQUE (user_a, user_b)
);

CREATE INDEX idx_matches_user_b ON matches (user_b, created_at DESC);
CREATE INDEX idx_matches_user_a ON matches (user_a, created_at DESC);

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT blocks_not_self CHECK (blocker_id <> blocked_id)
);

CREATE INDEX idx_blocks_blocked ON blocks (blocked_id, blocker_id);

CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT reports_reason_valid CHECK (reason IN ('spam', 'fake_profile', 'harassment', 'inappropriate_content', 'underage', 'other')),
  CONSTRAINT reports_status_valid CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT reports_details_len CHECK (char_length(details) <= 1000),
  CONSTRAINT reports_not_self CHECK (reporter_id <> reported_id)
);

CREATE INDEX idx_reports_status_created ON reports (status, created_at);
CREATE INDEX idx_reports_reported ON reports (reported_id);
