CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  position SMALLINT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_photos_user_position ON photos (user_id, position);
