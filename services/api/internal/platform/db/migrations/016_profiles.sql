CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coordinates are rounded to 0.01 degree (~1 km) by the API before storage.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX IF NOT EXISTS idx_profiles_discovery ON profiles (discoverable, gender);
CREATE INDEX IF NOT EXISTS idx_profiles_geo ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'non_binary']::text[]
  ),
  min_age SMALLINT NOT NULL CHECK (min_age >= 18),
  max_age SMALLINT NOT NULL CHECK (max_age <= 99),
  max_distance_km INTEGER NULL CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (min_age <= max_age)
);

CREATE TABLE IF NOT EXISTS interests (
  id SERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO interests (slug, label) VALUES
  ('voyage', 'Voyage'), ('cuisine', 'Cuisine'), ('cinema', 'Cinéma'), ('musique', 'Musique'),
  ('lecture', 'Lecture'), ('sport', 'Sport'), ('randonnee', 'Randonnée'), ('yoga', 'Yoga'),
  ('photo', 'Photographie'), ('jeux-video', 'Jeux vidéo'), ('jeux-de-societe', 'Jeux de société'),
  ('art', 'Art'), ('theatre', 'Théâtre'), ('concerts', 'Concerts'), ('danse', 'Danse'),
  ('animaux', 'Animaux'), ('jardinage', 'Jardinage'), ('technologie', 'Technologie'),
  ('cafe', 'Café'), ('vin', 'Vin'), ('running', 'Course à pied'), ('velo', 'Vélo'),
  ('plage', 'Plage'), ('montagne', 'Montagne'), ('bricolage', 'Bricolage'),
  ('benevolat', 'Bénévolat'), ('ecriture', 'Écriture'), ('meditation', 'Méditation')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id INTEGER NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);
CREATE INDEX IF NOT EXISTS idx_user_interests_interest ON user_interests (interest_id, user_id);

CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL CHECK (position >= 0),
  content_type TEXT NOT NULL DEFAULT 'image/jpeg',
  data BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
