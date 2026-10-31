CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position INT NOT NULL CHECK (position >= 0 AND position < 6),
  mime_type TEXT NOT NULL CHECK (mime_type = 'image/jpeg'),
  size_bytes INT NOT NULL CHECK (size_bytes > 0),
  data BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_photos_user_position ON photos (user_id, position);
