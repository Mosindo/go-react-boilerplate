CREATE TABLE IF NOT EXISTS blocks (
  blocker_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_user_id, blocked_user_id),
  CONSTRAINT chk_blocks_not_self CHECK (blocker_user_id <> blocked_user_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocked ON blocks (blocked_user_id, blocker_user_id);

CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_reports_reason CHECK (
    reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')
  ),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT chk_reports_details CHECK (char_length(details) <= 1000),
  CONSTRAINT chk_reports_not_self CHECK (reporter_user_id IS NULL OR reporter_user_id <> reported_user_id)
);

CREATE INDEX IF NOT EXISTS idx_reports_reported_created_at ON reports (reported_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_status_created_at ON reports (status, created_at DESC);
-- One open report per reporter/reported pair.
CREATE UNIQUE INDEX IF NOT EXISTS uq_reports_open_pair
  ON reports (reporter_user_id, reported_user_id) WHERE status = 'open';
