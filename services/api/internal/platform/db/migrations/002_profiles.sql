CREATE TABLE profiles (
  user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name  TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date  DATE NOT NULL,
  gender      TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'nonbinary')),
  bio         TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city        TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coordinates are rounded to 2 decimals (~1 km) by the API before storage.
  lat         DOUBLE PRECISION NULL CHECK (lat BETWEEN -90 AND 90),
  lng         DOUBLE PRECISION NULL CHECK (lng BETWEEN -180 AND 180),
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  discoverable  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((lat IS NULL) = (lng IS NULL))
);
CREATE INDEX idx_profiles_discoverable_geo ON profiles (lat, lng) WHERE discoverable;

CREATE TABLE preferences (
  user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in   TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'nonbinary']::TEXT[]
  ),
  age_min         SMALLINT NOT NULL DEFAULT 18 CHECK (age_min >= 18),
  age_max         SMALLINT NOT NULL DEFAULT 99 CHECK (age_max <= 120),
  max_distance_km INTEGER NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (age_min <= age_max)
);

CREATE TABLE interests (
  id    SMALLINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  slug  TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

CREATE TABLE user_interests (
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id);

CREATE TABLE photos (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position    SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  storage_key TEXT NOT NULL UNIQUE,
  width       INTEGER NOT NULL,
  height      INTEGER NOT NULL,
  bytes       INTEGER NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_photos_user_created_at ON photos (user_id, created_at);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyages'), ('cooking', 'Cuisine'), ('music', 'Musique'),
  ('cinema', 'Cinéma'), ('reading', 'Lecture'), ('hiking', 'Randonnée'),
  ('sports', 'Sport'), ('yoga', 'Yoga'), ('photography', 'Photographie'),
  ('gaming', 'Jeux vidéo'), ('art', 'Art'), ('dancing', 'Danse'),
  ('animals', 'Animaux'), ('coffee', 'Café'), ('wine', 'Vin'),
  ('tech', 'Technologie'), ('nature', 'Nature'), ('theatre', 'Théâtre'),
  ('cycling', 'Vélo'), ('languages', 'Langues');
