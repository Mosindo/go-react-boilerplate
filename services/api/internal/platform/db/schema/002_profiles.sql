-- Profiles, preferences, photos and interests.
CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  occupation TEXT NOT NULL DEFAULT '' CHECK (char_length(occupation) <= 80),
  looking_for TEXT NOT NULL DEFAULT 'open' CHECK (looking_for IN ('relationship', 'casual', 'friends', 'open')),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coordinates are snapped to a 0.01 degree grid (~1 km) before storage; never exact.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_birthdate ON profiles (birthdate);
CREATE INDEX idx_profiles_gender ON profiles (gender);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['man', 'woman', 'non_binary']
    CHECK (cardinality(interested_in) > 0 AND interested_in <@ ARRAY['man', 'woman', 'non_binary']::text[]),
  min_age INT NOT NULL DEFAULT 18 CHECK (min_age BETWEEN 18 AND 99),
  max_age INT NOT NULL DEFAULT 99 CHECK (max_age BETWEEN 18 AND 99),
  max_distance_km INT NULL CHECK (max_distance_km BETWEEN 1 AND 20000),
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (min_age <= max_age)
);

CREATE TABLE interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

CREATE TABLE user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id, user_id);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position INT NOT NULL CHECK (position >= 0),
  storage_key TEXT NOT NULL UNIQUE,
  width INT NOT NULL,
  height INT NOT NULL,
  size_bytes INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_photos_user_position ON photos (user_id, position, created_at);
CREATE INDEX idx_photos_user_created_at ON photos (user_id, created_at DESC);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyage'), ('cooking', 'Cuisine'), ('hiking', 'Randonnée'), ('music', 'Musique'),
  ('cinema', 'Cinéma'), ('reading', 'Lecture'), ('sport', 'Sport'), ('yoga', 'Yoga'),
  ('photography', 'Photographie'), ('art', 'Art'), ('gaming', 'Jeux vidéo'), ('dancing', 'Danse'),
  ('animals', 'Animaux'), ('coffee', 'Café'), ('wine', 'Vin'), ('nature', 'Nature'),
  ('technology', 'Technologie'), ('theatre', 'Théâtre'), ('running', 'Course à pied'), ('cycling', 'Vélo'),
  ('board_games', 'Jeux de société'), ('volunteering', 'Bénévolat'), ('languages', 'Langues'), ('fashion', 'Mode');

CREATE FUNCTION distance_km(lat1 DOUBLE PRECISION, lng1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lng2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION LANGUAGE SQL IMMUTABLE PARALLEL SAFE AS $$
  SELECT 6371.0 * 2 * asin(sqrt(least(1.0,
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lng2 - lng1) / 2), 2))))
$$;
