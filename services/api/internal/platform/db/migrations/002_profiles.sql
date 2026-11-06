-- Profiles: one row per user. Coordinates are stored rounded (about 1 km) and
-- are never returned by the API; clients only receive an approximate distance.
CREATE TABLE profiles (
  user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name       TEXT NOT NULL,
  birth_date       DATE NOT NULL,
  gender           TEXT NOT NULL,
  bio              TEXT NOT NULL DEFAULT '',
  city             TEXT NOT NULL DEFAULT '',
  lat              DOUBLE PRECISION NOT NULL,
  lng              DOUBLE PRECISION NOT NULL,
  is_discoverable  BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance    BOOLEAN NOT NULL DEFAULT TRUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_name CHECK (char_length(btrim(first_name)) BETWEEN 1 AND 40),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_adult CHECK (birth_date <= (CURRENT_DATE - INTERVAL '18 years')::date),
  CONSTRAINT chk_profiles_lat CHECK (lat BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_lng CHECK (lng BETWEEN -180 AND 180)
);
CREATE INDEX idx_profiles_discovery ON profiles (gender, lat, lng) WHERE is_discoverable;

CREATE TABLE preferences (
  user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in   TEXT[] NOT NULL DEFAULT ARRAY['woman', 'man', 'nonbinary'],
  min_age         SMALLINT NOT NULL DEFAULT 18,
  max_age         SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_pref_interested CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']
  ),
  CONSTRAINT chk_pref_age CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_pref_distance CHECK (max_distance_km BETWEEN 1 AND 500)
);

-- Photos are stored on the storage backend; position 0 is the main photo.
CREATE TABLE photos (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL,
  mime_type   TEXT NOT NULL,
  size_bytes  INTEGER NOT NULL,
  width       INTEGER NOT NULL,
  height      INTEGER NOT NULL,
  position    SMALLINT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT uq_photos_position UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);
CREATE UNIQUE INDEX idx_photos_storage_key ON photos (storage_key);

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
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id, user_id);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyage'), ('cooking', 'Cuisine'), ('hiking', 'Randonnée'), ('music', 'Musique'),
  ('cinema', 'Cinéma'), ('reading', 'Lecture'), ('sport', 'Sport'), ('yoga', 'Yoga'),
  ('photography', 'Photo'), ('art', 'Art'), ('gaming', 'Jeux vidéo'), ('dance', 'Danse'),
  ('animals', 'Animaux'), ('coffee', 'Café'), ('wine', 'Vin'), ('nature', 'Nature'),
  ('tech', 'Tech'), ('fashion', 'Mode'), ('festivals', 'Festivals'), ('boardgames', 'Jeux de société');
