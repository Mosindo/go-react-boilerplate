CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  position SMALLINT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT chk_photos_dimensions CHECK (width > 0 AND height > 0),
  CONSTRAINT chk_photos_size CHECK (size_bytes > 0)
);

-- Deferrable so that reordering can swap positions inside one transaction.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_photos_user_position') THEN
    ALTER TABLE photos
      ADD CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED;
  END IF;
END $$;

-- A profile is discoverable only when every mandatory field is set.
CREATE OR REPLACE FUNCTION refresh_profile_completeness(target_user_id UUID) RETURNS BOOLEAN AS $$
  UPDATE profiles p
  SET is_complete = (
        p.first_name <> ''
    AND p.birthdate IS NOT NULL
    AND p.gender IS NOT NULL
    AND p.latitude IS NOT NULL
    AND p.longitude IS NOT NULL
    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
    AND EXISTS (
      SELECT 1 FROM preferences pr
      WHERE pr.user_id = p.user_id AND cardinality(pr.interested_in) > 0
    )
  )
  WHERE p.user_id = target_user_id
  RETURNING is_complete;
$$ LANGUAGE sql;
