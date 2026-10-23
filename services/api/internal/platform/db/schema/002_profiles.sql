-- Profiles, matching preferences, interests.

CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL CHECK (char_length(city) BETWEEN 1 AND 80),
  -- Coarse location only (rounded by the service to ~2 km); never exposed to other users.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'non_binary']
  ),
  min_age SMALLINT NOT NULL DEFAULT 18 CHECK (min_age BETWEEN 18 AND 99),
  max_age SMALLINT NOT NULL DEFAULT 99 CHECK (max_age BETWEEN 18 AND 99),
  max_distance_km INTEGER NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (max_age >= min_age)
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

-- Reference data (not demo data): the catalogue of selectable interests.
INSERT INTO interests (slug, label) VALUES
  ('cinema', 'Cinéma'), ('series', 'Séries'), ('music', 'Musique'), ('concerts', 'Concerts'),
  ('reading', 'Lecture'), ('cooking', 'Cuisine'), ('restaurants', 'Restaurants'), ('wine', 'Vin'),
  ('travel', 'Voyages'), ('hiking', 'Randonnée'), ('running', 'Course à pied'), ('cycling', 'Vélo'),
  ('fitness', 'Fitness'), ('yoga', 'Yoga'), ('football', 'Football'), ('tennis', 'Tennis'),
  ('climbing', 'Escalade'), ('swimming', 'Natation'), ('photography', 'Photographie'), ('art', 'Art'),
  ('museums', 'Musées'), ('theatre', 'Théâtre'), ('dance', 'Danse'), ('gaming', 'Jeux vidéo'),
  ('boardgames', 'Jeux de société'), ('technology', 'Technologie'), ('nature', 'Nature'), ('animals', 'Animaux'),
  ('gardening', 'Jardinage'), ('volunteering', 'Bénévolat')
ON CONFLICT (slug) DO NOTHING;
