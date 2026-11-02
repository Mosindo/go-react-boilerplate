CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('woman', 'man', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coordinates are rounded to 0.01 degree (~1 km) by the service before storage.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_discovery ON profiles (gender, birth_date) WHERE discoverable;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'non_binary']::TEXT[]
  ),
  min_age INT NOT NULL DEFAULT 18 CHECK (min_age >= 18),
  max_age INT NOT NULL DEFAULT 99 CHECK (max_age <= 120),
  max_distance_km INT NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
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
  content_type TEXT NOT NULL DEFAULT 'image/jpeg',
  width INT NOT NULL,
  height INT NOT NULL,
  data BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_photos_user_position ON photos (user_id, position);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyage'), ('music', 'Musique'), ('cinema', 'Cinéma'), ('cooking', 'Cuisine'),
  ('hiking', 'Randonnée'), ('sport', 'Sport'), ('reading', 'Lecture'), ('photography', 'Photo'),
  ('art', 'Art'), ('gaming', 'Jeux vidéo'), ('dancing', 'Danse'), ('yoga', 'Yoga'),
  ('animals', 'Animaux'), ('coffee', 'Café'), ('wine', 'Vin'), ('tech', 'Technologie'),
  ('nature', 'Nature'), ('theatre', 'Théâtre'), ('festivals', 'Festivals'), ('languages', 'Langues');
CREATE INDEX idx_profiles_latitude ON profiles (latitude) WHERE discoverable AND latitude IS NOT NULL;
