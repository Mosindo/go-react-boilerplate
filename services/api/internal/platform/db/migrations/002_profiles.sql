CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coordinates are snapped to a ~1 km grid by the API before storage; never exact.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_discoverable ON profiles (gender, birth_date) WHERE is_visible;
CREATE INDEX idx_profiles_geo ON profiles (latitude, longitude) WHERE is_visible AND latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT '{}'
    CHECK (interested_in <@ ARRAY['man', 'woman', 'non_binary']::TEXT[]),
  min_age SMALLINT NOT NULL DEFAULT 18 CHECK (min_age >= 18),
  max_age SMALLINT NOT NULL DEFAULT 99 CHECK (max_age <= 120),
  max_distance_km INTEGER NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (min_age <= max_age)
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

INSERT INTO interests (slug, label) VALUES
  ('cuisine', 'Cuisine'), ('voyages', 'Voyages'), ('randonnee', 'Randonnée'),
  ('cinema', 'Cinéma'), ('musique', 'Musique'), ('concerts', 'Concerts'),
  ('lecture', 'Lecture'), ('sport', 'Sport'), ('yoga', 'Yoga'),
  ('photographie', 'Photographie'), ('art', 'Art'), ('jeux-video', 'Jeux vidéo'),
  ('jeux-de-societe', 'Jeux de société'), ('danse', 'Danse'), ('theatre', 'Théâtre'),
  ('nature', 'Nature'), ('animaux', 'Animaux'), ('cafe', 'Café'),
  ('gastronomie', 'Gastronomie'), ('vin', 'Vin'), ('velo', 'Vélo'),
  ('running', 'Course à pied'), ('natation', 'Natation'), ('bricolage', 'Bricolage'),
  ('jardinage', 'Jardinage'), ('technologie', 'Technologie'), ('mode', 'Mode'),
  ('benevolat', 'Bénévolat');

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  thumb_key TEXT NOT NULL UNIQUE,
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);
CREATE INDEX idx_photos_user_created_at ON photos (user_id, created_at);
