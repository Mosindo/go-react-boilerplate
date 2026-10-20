CREATE TABLE IF NOT EXISTS votes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_type TEXT NOT NULL,
  target_id UUID NOT NULL,
  value SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_votes_value CHECK (value IN (-1, 1))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_user_target
  ON votes (user_id, target_type, target_id);

CREATE INDEX IF NOT EXISTS idx_votes_target
  ON votes (target_type, target_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_votes_user_created_at
  ON votes (user_id, created_at DESC, id DESC);
