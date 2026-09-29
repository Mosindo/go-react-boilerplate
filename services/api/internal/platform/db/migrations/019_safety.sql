CREATE TABLE IF NOT EXISTS blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT chk_blocks_distinct CHECK (blocker_id <> blocked_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocked ON blocks (blocked_id, blocker_id);

CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_reports_reason CHECK (reason IN ('spam', 'fake_profile', 'harassment', 'inappropriate_content', 'underage', 'scam', 'other')),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT chk_reports_details CHECK (char_length(details) <= 1000),
  CONSTRAINT chk_reports_distinct CHECK (reporter_id <> reported_id)
);

CREATE INDEX IF NOT EXISTS idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_status_created ON reports (status, created_at DESC);
