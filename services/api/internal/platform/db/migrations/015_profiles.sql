-- Profiles, preferences, interests and photos.

CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('woman', 'man', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  occupation TEXT NOT NULL DEFAULT '' CHECK (char_length(occupation) <= 60),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coarse coordinates only (rounded to 2 decimals, ~1 km) and never exposed to other users.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  location_updated_at TIMESTAMPTZ NULL,
  is_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_profiles_discovery
  ON profiles (gender, latitude, longitude)
  WHERE is_discoverable;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'non_binary']
  ),
  min_age SMALLINT NOT NULL DEFAULT 18 CHECK (min_age >= 18),
  max_age SMALLINT NOT NULL DEFAULT 99 CHECK (max_age <= 120),
  max_distance_km INTEGER NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (min_age <= max_age)
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyage'), ('cooking', 'Cuisine'), ('hiking', 'Randonnée'), ('running', 'Course à pied'),
  ('fitness', 'Fitness'), ('yoga', 'Yoga'), ('cycling', 'Vélo'), ('swimming', 'Natation'),
  ('football', 'Football'), ('tennis', 'Tennis'), ('climbing', 'Escalade'), ('music', 'Musique'),
  ('concerts', 'Concerts'), ('dancing', 'Danse'), ('cinema', 'Cinéma'), ('series', 'Séries'),
  ('reading', 'Lecture'), ('writing', 'Écriture'), ('photography', 'Photographie'), ('art', 'Art'),
  ('museums', 'Musées'), ('theatre', 'Théâtre'), ('gaming', 'Jeux vidéo'), ('board_games', 'Jeux de société'),
  ('technology', 'Technologie'), ('nature', 'Nature'), ('animals', 'Animaux'), ('gardening', 'Jardinage'),
  ('coffee', 'Café'), ('wine', 'Vin'), ('food', 'Gastronomie'), ('fashion', 'Mode'),
  ('volunteering', 'Bénévolat'), ('languages', 'Langues'), ('meditation', 'Méditation'), ('road_trips', 'Road trips')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest ON user_interests (interest_id, user_id);

CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  content_type TEXT NOT NULL,
  width INTEGER NOT NULL CHECK (width > 0),
  height INTEGER NOT NULL CHECK (height > 0),
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- Distance between two points in kilometres (haversine, spherical Earth).
CREATE OR REPLACE FUNCTION haversine_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE SQL IMMUTABLE PARALLEL SAFE AS $$
  SELECT 6371.0 * 2 * asin(least(1.0, sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  )))
$$;
