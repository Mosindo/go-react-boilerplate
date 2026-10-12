CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('woman', 'man', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  occupation TEXT NOT NULL DEFAULT '' CHECK (char_length(occupation) <= 80),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coarse location only: coordinates are rounded to 0.01 degree (~1 km) before storage.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  location_updated_at TIMESTAMPTZ NULL,
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT '{}',
  age_min SMALLINT NOT NULL DEFAULT 18 CHECK (age_min BETWEEN 18 AND 99),
  age_max SMALLINT NOT NULL DEFAULT 99 CHECK (age_max BETWEEN 18 AND 99),
  max_distance_km INTEGER NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 20000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (age_min <= age_max),
  CHECK (interested_in <@ ARRAY['woman', 'man', 'non_binary']::TEXT[])
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
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL CHECK (position >= 0),
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_photos_user_position ON photos (user_id, position);

CREATE INDEX idx_profiles_discovery ON profiles (gender, birth_date) WHERE is_visible;

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyage'), ('music', 'Musique'), ('cinema', 'Cinéma'), ('cooking', 'Cuisine'),
  ('sport', 'Sport'), ('hiking', 'Randonnée'), ('reading', 'Lecture'), ('photography', 'Photo'),
  ('gaming', 'Jeux vidéo'), ('art', 'Art'), ('dance', 'Danse'), ('yoga', 'Yoga'),
  ('pets', 'Animaux'), ('coffee', 'Café'), ('wine', 'Vin'), ('theatre', 'Théâtre'),
  ('tech', 'Technologie'), ('nature', 'Nature'), ('fitness', 'Fitness'), ('festivals', 'Festivals'),
  ('boardgames', 'Jeux de société'), ('fashion', 'Mode'), ('volunteering', 'Bénévolat'), ('languages', 'Langues')
ON CONFLICT (slug) DO NOTHING;

CREATE OR REPLACE FUNCTION haversine_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION AS $$
  SELECT 2 * 6371 * asin(sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  ))
$$ LANGUAGE SQL IMMUTABLE;
