CREATE TABLE profiles (
  user_id       UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name    TEXT NOT NULL,
  birth_date    DATE NOT NULL,
  gender        TEXT NULL,
  bio           TEXT NOT NULL DEFAULT '',
  city          TEXT NOT NULL DEFAULT '',
  -- Coordinates are deliberately rounded (2 decimals, ~1 km) before storage.
  latitude      DOUBLE PRECISION NULL,
  longitude     DOUBLE PRECISION NULL,
  discoverable  BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_gender_check CHECK (gender IS NULL OR gender IN ('man', 'woman', 'non_binary')),
  CONSTRAINT profiles_first_name_len CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT profiles_bio_len CHECK (char_length(bio) <= 500),
  CONSTRAINT profiles_lat_range CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT profiles_lng_range CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_discovery ON profiles (gender, birth_date) WHERE discoverable;
CREATE INDEX idx_profiles_geo ON profiles (latitude, longitude) WHERE discoverable;

CREATE TABLE preferences (
  user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in   TEXT[] NOT NULL DEFAULT '{}',
  age_min         SMALLINT NOT NULL DEFAULT 18,
  age_max         SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (age_min >= 18 AND age_max <= 99 AND age_min <= age_max),
  CONSTRAINT preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500),
  CONSTRAINT preferences_interested_in CHECK (interested_in <@ ARRAY['man', 'woman', 'non_binary']::TEXT[])
);

CREATE TABLE photos (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  position    SMALLINT NOT NULL,
  width       INTEGER NOT NULL,
  height      INTEGER NOT NULL,
  size_bytes  INTEGER NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_position_range CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);
CREATE INDEX idx_photos_user_created ON photos (user_id, created_at);

CREATE TABLE interests (
  id    SMALLSERIAL PRIMARY KEY,
  slug  TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

CREATE TABLE user_interests (
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id);

INSERT INTO interests (slug, label) VALUES
  ('hiking', 'Randonnée'), ('cooking', 'Cuisine'), ('travel', 'Voyage'), ('music', 'Musique'),
  ('cinema', 'Cinéma'), ('reading', 'Lecture'), ('sport', 'Sport'), ('yoga', 'Yoga'),
  ('photography', 'Photo'), ('gaming', 'Jeux vidéo'), ('art', 'Art'), ('dancing', 'Danse'),
  ('coffee', 'Café'), ('wine', 'Vin'), ('animals', 'Animaux'), ('nature', 'Nature'),
  ('tech', 'Tech'), ('festivals', 'Festivals'), ('boardgames', 'Jeux de société'), ('running', 'Course à pied'),
  ('cycling', 'Vélo'), ('theatre', 'Théâtre'), ('languages', 'Langues'), ('volunteering', 'Bénévolat');
