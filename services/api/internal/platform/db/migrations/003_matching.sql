-- One row per decision a user takes on another profile. A decision is final:
-- a profile that has been liked or passed is never proposed again.
CREATE TABLE swipes (
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (from_user_id, to_user_id),
  CONSTRAINT swipes_not_self CHECK (from_user_id <> to_user_id)
);
CREATE INDEX idx_swipes_to_likes ON swipes (to_user_id, created_at DESC) WHERE action = 'like';

-- A match is also the conversation between its two users (1:1).
-- user_a_id < user_b_id guarantees one row per pair.
CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  a_hidden_at TIMESTAMPTZ NULL,
  b_hidden_at TIMESTAMPTZ NULL,
  last_message_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT matches_ordered CHECK (user_a_id < user_b_id),
  CONSTRAINT matches_pair_unique UNIQUE (user_a_id, user_b_id)
);
CREATE INDEX idx_matches_user_a ON matches (user_a_id, COALESCE(last_message_at, created_at) DESC);
CREATE INDEX idx_matches_user_b ON matches (user_b_id, COALESCE(last_message_at, created_at) DESC);

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT blocks_not_self CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id, blocker_id);

-- Reports outlive the reported account (reported_id becomes NULL) so abuse
-- handling keeps its evidence; they are removed with the reporter's account.
CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL CHECK (reason IN ('spam', 'fake_profile', 'harassment', 'inappropriate_content', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE INDEX idx_reports_status ON reports (status, created_at DESC);
