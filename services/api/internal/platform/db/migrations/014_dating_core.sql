-- Dating domain: accounts, profiles, preferences, interests, photos.
-- Every statement is idempotent: migrations are re-applied at each boot.

ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_users_status') THEN
    ALTER TABLE users ADD CONSTRAINT chk_users_status CHECK (status IN ('active', 'suspended'));
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_users_role') THEN
    ALTER TABLE users ADD CONSTRAINT chk_users_role CHECK (role IN ('user', 'admin'));
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (lower(email));

CREATE TABLE IF NOT EXISTS password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user ON password_reset_tokens (user_id, created_at DESC);

-- Coordinates are stored rounded to 0.01 degree (~1 km) and never exposed by the API.
CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'non_binary', 'other')),
  CONSTRAINT chk_profiles_first_name CHECK (char_length(first_name) BETWEEN 1 AND 50),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_city CHECK (char_length(city) <= 80),
  CONSTRAINT chk_profiles_lat CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_lng CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX IF NOT EXISTS idx_profiles_discovery ON profiles (discoverable, gender, birth_date);
CREATE INDEX IF NOT EXISTS idx_profiles_location ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['woman', 'man', 'non_binary', 'other'],
  age_min SMALLINT NOT NULL DEFAULT 18,
  age_max SMALLINT NOT NULL DEFAULT 99,
  max_distance_km INTEGER NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_age CHECK (age_min >= 18 AND age_max <= 99 AND age_min <= age_max),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 20000),
  CONSTRAINT chk_preferences_interested CHECK (
    cardinality(interested_in) BETWEEN 1 AND 4
    AND interested_in <@ ARRAY['woman', 'man', 'non_binary', 'other']
  )
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);
CREATE INDEX IF NOT EXISTS idx_user_interests_interest ON user_interests (interest_id, user_id);

INSERT INTO interests (slug, label) VALUES
  ('hiking', 'Randonnée'), ('cooking', 'Cuisine'), ('travel', 'Voyages'), ('music', 'Musique'),
  ('concerts', 'Concerts'), ('cinema', 'Cinéma'), ('reading', 'Lecture'), ('photography', 'Photographie'),
  ('sports', 'Sport'), ('running', 'Course à pied'), ('yoga', 'Yoga'), ('climbing', 'Escalade'),
  ('cycling', 'Vélo'), ('gaming', 'Jeux vidéo'), ('board_games', 'Jeux de société'), ('art', 'Art'),
  ('museums', 'Musées'), ('theatre', 'Théâtre'), ('dancing', 'Danse'), ('animals', 'Animaux'),
  ('gardening', 'Jardinage'), ('coffee', 'Café'), ('wine', 'Vin'), ('tech', 'Technologie'),
  ('volunteering', 'Bénévolat'), ('languages', 'Langues')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL,
  storage_key TEXT NOT NULL UNIQUE,
  mime_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 8),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);
