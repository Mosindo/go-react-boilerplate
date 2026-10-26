-- Photos are re-encoded server side (JPEG, EXIF stripped) and stored in the
-- database so that backups, access control and account deletion stay consistent.
CREATE TABLE photos (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position     INT NOT NULL CHECK (position BETWEEN 0 AND 5),
  content_type TEXT NOT NULL DEFAULT 'image/jpeg',
  width        INT NOT NULL,
  height       INT NOT NULL,
  data         BYTEA NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);
