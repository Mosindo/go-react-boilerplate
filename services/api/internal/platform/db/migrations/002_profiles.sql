-- Profile: public-facing data. Coordinates are stored already rounded (~1 km grid)
-- by the API; exact positions are never persisted.
CREATE TABLE profiles (
  user_id        UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name     TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date     DATE NOT NULL,
  gender         TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'nonbinary')),
  bio            TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  job_title      TEXT NOT NULL DEFAULT '' CHECK (char_length(job_title) <= 80),
  city           TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  latitude       DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude      DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  is_visible     BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance  BOOLEAN NOT NULL DEFAULT TRUE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_discovery ON profiles (gender, birth_date) WHERE is_visible;
CREATE INDEX idx_profiles_last_active ON profiles (last_active_at DESC);

CREATE TABLE user_preferences (
  user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in   TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'nonbinary']
  ),
  min_age         INT NOT NULL DEFAULT 18 CHECK (min_age BETWEEN 18 AND 99),
  max_age         INT NOT NULL DEFAULT 99 CHECK (max_age BETWEEN 18 AND 99),
  max_distance_km INT NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 20000),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT user_preferences_age_range CHECK (min_age <= max_age)
);

CREATE TABLE interests (
  id    SMALLSERIAL PRIMARY KEY,
  slug  TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

CREATE TABLE user_interests (
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyages'), ('hiking', 'Randonnée'), ('cooking', 'Cuisine'), ('wine', 'Vin'),
  ('coffee', 'Café'), ('cinema', 'Cinéma'), ('series', 'Séries'), ('music', 'Musique'),
  ('concerts', 'Concerts'), ('reading', 'Lecture'), ('art', 'Art'), ('photography', 'Photographie'),
  ('yoga', 'Yoga'), ('running', 'Course à pied'), ('cycling', 'Vélo'), ('fitness', 'Sport'),
  ('climbing', 'Escalade'), ('swimming', 'Natation'), ('football', 'Football'), ('dancing', 'Danse'),
  ('gaming', 'Jeux vidéo'), ('boardgames', 'Jeux de société'), ('tech', 'Technologie'), ('nature', 'Nature'),
  ('animals', 'Animaux'), ('gardening', 'Jardinage'), ('fashion', 'Mode'), ('theatre', 'Théâtre'),
  ('volunteering', 'Bénévolat'), ('languages', 'Langues')
ON CONFLICT (slug) DO NOTHING;

-- Great-circle distance in kilometres.
CREATE OR REPLACE FUNCTION haversine_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION LANGUAGE SQL IMMUTABLE PARALLEL SAFE AS $$
  SELECT 12742.0 * asin(sqrt(least(1.0,
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  )))
$$;

CREATE INDEX idx_profiles_latitude ON profiles (latitude) WHERE is_visible AND latitude IS NOT NULL;
