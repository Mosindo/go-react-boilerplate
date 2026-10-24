CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('woman', 'man', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coarse coordinates only (rounded to 2 decimals, ~1 km) to limit stored location precision.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_profiles_gender_birth_date
  ON profiles (gender, birth_date)
  WHERE discoverable;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL
    CHECK (cardinality(interested_in) BETWEEN 1 AND 3
           AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]),
  age_min INT NOT NULL CHECK (age_min BETWEEN 18 AND 99),
  age_max INT NOT NULL CHECK (age_max BETWEEN 18 AND 99),
  -- NULL means "anywhere".
  max_distance_km INT NULL CHECK (max_distance_km BETWEEN 1 AND 20000),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (age_max >= age_min)
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

-- Reference catalog (not demo data).
INSERT INTO interests (slug, label) VALUES
  ('voyage', 'Voyage'), ('cuisine', 'Cuisine'), ('cinema', 'Cinéma'), ('musique', 'Musique'),
  ('concerts', 'Concerts'), ('lecture', 'Lecture'), ('randonnee', 'Randonnée'), ('sport', 'Sport'),
  ('yoga', 'Yoga'), ('course', 'Course à pied'), ('velo', 'Vélo'), ('photo', 'Photographie'),
  ('art', 'Art'), ('theatre', 'Théâtre'), ('jeux-video', 'Jeux vidéo'), ('jeux-societe', 'Jeux de société'),
  ('animaux', 'Animaux'), ('nature', 'Nature'), ('cafe', 'Café'), ('vin', 'Vin'),
  ('bricolage', 'Bricolage'), ('jardinage', 'Jardinage'), ('danse', 'Danse'), ('technologie', 'Technologie'),
  ('mode', 'Mode'), ('benevolat', 'Bénévolat'), ('plage', 'Plage'), ('montagne', 'Montagne')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);

CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  storage_key TEXT NOT NULL UNIQUE,
  mime_type TEXT NOT NULL,
  size_bytes INT NOT NULL CHECK (size_bytes > 0),
  width INT NOT NULL CHECK (width > 0),
  height INT NOT NULL CHECK (height > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_photos_user_position ON photos (user_id, position);
DROP TABLE IF EXISTS files;
