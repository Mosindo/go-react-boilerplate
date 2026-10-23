-- Photos: metadata is separate from the bytes so list queries never read blobs.
-- A user's photos live in the database so account deletion removes them atomically.

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  width INTEGER NOT NULL CHECK (width > 0),
  height INTEGER NOT NULL CHECK (height > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);

CREATE TABLE photo_blobs (
  photo_id UUID PRIMARY KEY REFERENCES photos(id) ON DELETE CASCADE,
  full_data BYTEA NOT NULL,
  thumb_data BYTEA NOT NULL
);
