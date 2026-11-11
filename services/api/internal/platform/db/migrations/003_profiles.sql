CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('woman', 'man', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  job_title TEXT NOT NULL DEFAULT '' CHECK (char_length(job_title) <= 60),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 60),
  -- Coordinates are rounded server-side (~1 km grid) before being stored.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  location_updated_at TIMESTAMPTZ NULL,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  is_demo BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX idx_profiles_discoverable_active ON profiles (last_active_at DESC) WHERE discoverable;
CREATE INDEX idx_profiles_location ON profiles (latitude, longitude) WHERE discoverable AND latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  ),
  age_min SMALLINT NOT NULL DEFAULT 18 CHECK (age_min BETWEEN 18 AND 99),
  age_max SMALLINT NOT NULL DEFAULT 99 CHECK (age_max BETWEEN 18 AND 99),
  max_distance_km SMALLINT NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (age_min <= age_max)
);

CREATE TABLE interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO interests (slug, label) VALUES
  ('hiking', 'Randonnée'),
  ('cooking', 'Cuisine'),
  ('travel', 'Voyages'),
  ('music', 'Musique'),
  ('concerts', 'Concerts'),
  ('cinema', 'Cinéma'),
  ('series', 'Séries'),
  ('reading', 'Lecture'),
  ('photography', 'Photo'),
  ('art', 'Art & expos'),
  ('theatre', 'Théâtre'),
  ('dance', 'Danse'),
  ('yoga', 'Yoga'),
  ('running', 'Course à pied'),
  ('cycling', 'Vélo'),
  ('climbing', 'Escalade'),
  ('swimming', 'Natation'),
  ('football', 'Football'),
  ('gaming', 'Jeux vidéo'),
  ('board_games', 'Jeux de société'),
  ('wine', 'Vin'),
  ('coffee', 'Café'),
  ('brunch', 'Brunch'),
  ('vegetarian', 'Végétarien'),
  ('gardening', 'Jardinage'),
  ('animals', 'Animaux'),
  ('volunteering', 'Bénévolat'),
  ('tech', 'Tech'),
  ('languages', 'Langues'),
  ('meditation', 'Méditation')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX idx_user_interests_interest ON user_interests (interest_id, user_id);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  width INTEGER NOT NULL CHECK (width > 0),
  height INTEGER NOT NULL CHECK (height > 0),
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);

CREATE INDEX idx_photos_user_created_at ON photos (user_id, created_at DESC);
