CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_id <> target_id)
);

CREATE INDEX idx_swipes_target_action ON swipes (target_id, action);

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- The pair is stored in canonical order so a pair can only match once.
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ NULL,
  ended_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_matches_order CHECK (user_a_id < user_b_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX idx_matches_user_b ON matches (user_b_id);

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT chk_blocks_not_self CHECK (blocker_id <> blocked_id)
);

CREATE INDEX idx_blocks_blocked ON blocks (blocked_id);

CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  -- Kept (nullable) when the reported account is deleted so moderation keeps a trail.
  reported_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_reports_reason CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT chk_reports_details CHECK (length(details) <= 1000)
);

CREATE INDEX idx_reports_status_created ON reports (status, created_at DESC);
CREATE INDEX idx_reports_reported ON reports (reported_id);
CREATE INDEX idx_reports_reporter ON reports (reporter_id, created_at DESC);
