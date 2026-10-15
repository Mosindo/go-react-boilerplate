CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  -- Coarse location only (rounded by the API to 2 decimals, roughly 1 km).
  city TEXT NULL CHECK (city IS NULL OR char_length(city) <= 80),
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_profiles_discoverable
  ON profiles (gender, birth_date)
  WHERE is_visible;

CREATE INDEX IF NOT EXISTS idx_profiles_latitude
  ON profiles (latitude)
  WHERE is_visible AND latitude IS NOT NULL;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  min_age INTEGER NOT NULL DEFAULT 18 CHECK (min_age >= 18),
  max_age INTEGER NOT NULL DEFAULT 99 CHECK (max_age <= 99),
  max_distance_km INTEGER NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (min_age <= max_age),
  CONSTRAINT preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'nonbinary']::TEXT[]
  )
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]+$'),
  label TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest
  ON user_interests (interest_id, user_id);

-- Interest catalog (reference data, not demo users).
INSERT INTO interests (slug, label) VALUES
  ('voyages', 'Voyages'),
  ('cuisine', 'Cuisine'),
  ('randonnee', 'Randonnée'),
  ('cinema', 'Cinéma'),
  ('musique', 'Musique'),
  ('concerts', 'Concerts'),
  ('lecture', 'Lecture'),
  ('sport', 'Sport'),
  ('yoga', 'Yoga'),
  ('course-a-pied', 'Course à pied'),
  ('velo', 'Vélo'),
  ('photographie', 'Photographie'),
  ('art', 'Art'),
  ('theatre', 'Théâtre'),
  ('jeux-video', 'Jeux vidéo'),
  ('jeux-de-societe', 'Jeux de société'),
  ('animaux', 'Animaux'),
  ('nature', 'Nature'),
  ('jardinage', 'Jardinage'),
  ('cafe', 'Café'),
  ('vin', 'Vin'),
  ('danse', 'Danse'),
  ('technologie', 'Technologie'),
  ('bricolage', 'Bricolage'),
  ('mode', 'Mode'),
  ('benevolat', 'Bénévolat')
ON CONFLICT (slug) DO NOTHING;
