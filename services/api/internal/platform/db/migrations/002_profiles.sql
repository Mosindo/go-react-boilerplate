CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  -- Coarse coordinates only: the API rounds to 2 decimals (~1 km) before storing.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_first_name_length CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT profiles_gender_valid CHECK (gender IN ('man', 'woman', 'non_binary')),
  CONSTRAINT profiles_bio_length CHECK (char_length(bio) <= 500),
  CONSTRAINT profiles_city_length CHECK (char_length(city) <= 80),
  CONSTRAINT profiles_latitude_range CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT profiles_longitude_range CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_discovery ON profiles (gender, birth_date) WHERE is_visible;
CREATE INDEX idx_profiles_location ON profiles (latitude, longitude) WHERE is_visible;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km INTEGER NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_interested_in_valid CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'non_binary']::TEXT[]
  ),
  CONSTRAINT preferences_age_range CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT preferences_distance_range CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  position SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_position_range CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT photos_user_position_unique UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE UNIQUE INDEX idx_photos_storage_key ON photos (storage_key);

CREATE TABLE interests (
  id SMALLINT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

CREATE TABLE user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id, user_id);

INSERT INTO interests (id, slug, label) VALUES
  (1, 'travel', 'Voyages'),
  (2, 'cooking', 'Cuisine'),
  (3, 'music', 'Musique'),
  (4, 'movies', 'Cinéma'),
  (5, 'books', 'Lecture'),
  (6, 'hiking', 'Randonnée'),
  (7, 'sport', 'Sport'),
  (8, 'yoga', 'Yoga'),
  (9, 'photography', 'Photographie'),
  (10, 'art', 'Art'),
  (11, 'gaming', 'Jeux vidéo'),
  (12, 'board_games', 'Jeux de société'),
  (13, 'dancing', 'Danse'),
  (14, 'concerts', 'Concerts'),
  (15, 'coffee', 'Café'),
  (16, 'wine', 'Vin'),
  (17, 'animals', 'Animaux'),
  (18, 'nature', 'Nature'),
  (19, 'technology', 'Technologie'),
  (20, 'fashion', 'Mode'),
  (21, 'theatre', 'Théâtre'),
  (22, 'running', 'Course à pied'),
  (23, 'cycling', 'Vélo'),
  (24, 'volunteering', 'Bénévolat');

-- Great-circle distance in kilometres between two coordinates.
CREATE FUNCTION haversine_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE SQL IMMUTABLE PARALLEL SAFE
AS $$
  SELECT 6371.0 * 2 * asin(least(1.0, sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  )))
$$;
