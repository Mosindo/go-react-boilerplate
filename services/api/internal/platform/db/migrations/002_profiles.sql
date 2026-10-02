CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 50),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('woman', 'man', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coarse coordinates only (rounded to 2 decimals, about 1 km). Never exposed to other users.
  latitude DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_profiles_discovery ON profiles (gender, latitude, longitude) WHERE discoverable;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL CHECK (cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']),
  min_age INT NOT NULL DEFAULT 18 CHECK (min_age >= 18),
  max_age INT NOT NULL DEFAULT 99 CHECK (max_age <= 120),
  max_distance_km INT NULL CHECK (max_distance_km IS NULL OR max_distance_km BETWEEN 1 AND 20000),
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
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL CHECK (position >= 0),
  storage_key TEXT NOT NULL UNIQUE,
  content_type TEXT NOT NULL,
  size_bytes INT NOT NULL CHECK (size_bytes > 0),
  width INT NOT NULL,
  height INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_photos_user_created ON photos (user_id, created_at);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyage'), ('cooking', 'Cuisine'), ('hiking', 'Randonnée'), ('music', 'Musique'),
  ('cinema', 'Cinéma'), ('reading', 'Lecture'), ('sport', 'Sport'), ('yoga', 'Yoga'),
  ('photography', 'Photographie'), ('art', 'Art'), ('gaming', 'Jeux vidéo'), ('dancing', 'Danse'),
  ('animals', 'Animaux'), ('coffee', 'Café'), ('wine', 'Vin'), ('nature', 'Nature'),
  ('tech', 'Technologie'), ('theatre', 'Théâtre'), ('running', 'Course à pied'), ('cycling', 'Vélo'),
  ('festivals', 'Festivals'), ('board_games', 'Jeux de société'), ('volunteering', 'Bénévolat'), ('languages', 'Langues');
