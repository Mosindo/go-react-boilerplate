CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  gender TEXT NULL CHECK (gender IN ('woman', 'man', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coordinates are rounded to 2 decimals (~1 km) before being stored.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  is_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_geo_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT '{}',
  age_min INT NOT NULL DEFAULT 18 CHECK (age_min >= 18),
  age_max INT NOT NULL DEFAULT 99 CHECK (age_max <= 120),
  max_distance_km INT NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (age_min <= age_max),
  CONSTRAINT preferences_interested_in CHECK (interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[])
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

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyage'), ('cooking', 'Cuisine'), ('music', 'Musique'), ('cinema', 'Cinéma'),
  ('reading', 'Lecture'), ('sport', 'Sport'), ('hiking', 'Randonnée'), ('photography', 'Photo'),
  ('art', 'Art'), ('gaming', 'Jeux vidéo'), ('yoga', 'Yoga'), ('dancing', 'Danse'),
  ('animals', 'Animaux'), ('nature', 'Nature'), ('coffee', 'Café'), ('wine', 'Vin'),
  ('tech', 'Tech'), ('theatre', 'Théâtre'), ('festivals', 'Festivals'), ('volunteering', 'Bénévolat');

CREATE INDEX idx_profiles_discover ON profiles (gender, is_discoverable);
CREATE INDEX idx_profiles_geo ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;
