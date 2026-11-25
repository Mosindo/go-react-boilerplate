CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NULL,
  city TEXT NULL,
  height_cm SMALLINT NULL,
  looking_for TEXT NULL,
  -- Coordinates are rounded to 2 decimals (~1 km) before storage. Exact GPS
  -- positions are never persisted nor returned by the API.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_first_name CHECK (length(btrim(first_name)) BETWEEN 1 AND 40),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_bio CHECK (length(bio) <= 500),
  CONSTRAINT chk_profiles_job CHECK (job_title IS NULL OR length(job_title) <= 60),
  CONSTRAINT chk_profiles_city CHECK (city IS NULL OR length(city) <= 60),
  CONSTRAINT chk_profiles_height CHECK (height_cm IS NULL OR height_cm BETWEEN 120 AND 230),
  CONSTRAINT chk_profiles_looking_for CHECK (looking_for IS NULL OR looking_for IN ('relationship', 'casual', 'friendship', 'unsure')),
  CONSTRAINT chk_profiles_latitude CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_longitude CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL)),
  CONSTRAINT chk_profiles_adult CHECK (birthdate <= (CURRENT_DATE - INTERVAL '18 years'))
);

CREATE INDEX idx_profiles_discovery ON profiles (gender, birthdate) WHERE discoverable AND latitude IS NOT NULL;
CREATE INDEX idx_profiles_latitude ON profiles (latitude) WHERE discoverable AND latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT '{}',
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]),
  CONSTRAINT chk_preferences_ages CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

-- Reference catalog (product data, not demo data).
INSERT INTO interests (slug, label) VALUES
  ('cooking', 'Cuisine'),
  ('travel', 'Voyages'),
  ('hiking', 'Randonnée'),
  ('running', 'Course à pied'),
  ('yoga', 'Yoga'),
  ('climbing', 'Escalade'),
  ('cycling', 'Vélo'),
  ('swimming', 'Natation'),
  ('football', 'Football'),
  ('dance', 'Danse'),
  ('music', 'Musique'),
  ('concerts', 'Concerts'),
  ('cinema', 'Cinéma'),
  ('series', 'Séries'),
  ('theatre', 'Théâtre'),
  ('reading', 'Lecture'),
  ('writing', 'Écriture'),
  ('photography', 'Photographie'),
  ('painting', 'Peinture'),
  ('museums', 'Musées'),
  ('video_games', 'Jeux vidéo'),
  ('board_games', 'Jeux de société'),
  ('tech', 'Tech'),
  ('science', 'Sciences'),
  ('gardening', 'Jardinage'),
  ('animals', 'Animaux'),
  ('wine', 'Vin'),
  ('coffee', 'Café'),
  ('brunch', 'Brunch'),
  ('vegan', 'Cuisine végétale'),
  ('volunteering', 'Bénévolat'),
  ('meditation', 'Méditation'),
  ('fashion', 'Mode'),
  ('languages', 'Langues'),
  ('nature', 'Nature'),
  ('sea', 'Mer');

CREATE TABLE user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX idx_user_interests_interest ON user_interests (interest_id);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL,
  position SMALLINT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  mime_type TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT chk_photos_mime CHECK (mime_type IN ('image/jpeg')),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE UNIQUE INDEX idx_photos_storage_key ON photos (storage_key);
CREATE INDEX idx_photos_user_position ON photos (user_id, position);

-- Great-circle distance in kilometres (haversine).
CREATE FUNCTION distance_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE SQL IMMUTABLE STRICT PARALLEL SAFE AS $$
  SELECT 6371.0 * 2 * asin(least(1.0, sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  )))
$$;
