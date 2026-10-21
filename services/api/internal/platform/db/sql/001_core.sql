-- Alma core schema: accounts, profiles, photos, preferences, interests.
-- Deleting a user cascades to every personal row (right to erasure).

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  birth_date    DATE NOT NULL,
  status        TEXT NOT NULL DEFAULT 'active',
  is_demo       BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT users_email_lower CHECK (email = lower(email)),
  CONSTRAINT users_status_valid CHECK (status IN ('active', 'suspended'))
);
CREATE UNIQUE INDEX idx_users_email ON users (email);

CREATE TABLE sessions (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash   TEXT NOT NULL,
  user_agent   TEXT NULL,
  ip_address   INET NULL,
  expires_at   TIMESTAMPTZ NOT NULL,
  revoked_at   TIMESTAMPTZ NULL,
  last_used_at TIMESTAMPTZ NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions (token_hash);
CREATE INDEX idx_sessions_user ON sessions (user_id, created_at DESC);

CREATE TABLE password_resets (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at    TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_password_resets_user ON password_resets (user_id, created_at DESC);

CREATE TABLE profiles (
  user_id      UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name   TEXT NOT NULL,
  gender       TEXT NOT NULL,
  bio          TEXT NOT NULL DEFAULT '',
  city         TEXT NOT NULL DEFAULT '',
  -- Coordinates are rounded to 2 decimals (~1 km) before storage: the exact position is never kept.
  latitude     DOUBLE PRECISION NULL,
  longitude    DOUBLE PRECISION NULL,
  occupation   TEXT NOT NULL DEFAULT '',
  height_cm    SMALLINT NULL,
  looking_for  TEXT NOT NULL DEFAULT 'unsure',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_gender_valid CHECK (gender IN ('man', 'woman', 'non_binary')),
  CONSTRAINT profiles_looking_for_valid CHECK (looking_for IN ('long_term', 'short_term', 'friendship', 'open', 'unsure')),
  CONSTRAINT profiles_first_name_len CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT profiles_bio_len CHECK (char_length(bio) <= 500),
  CONSTRAINT profiles_height_range CHECK (height_cm IS NULL OR height_cm BETWEEN 100 AND 250),
  CONSTRAINT profiles_lat_range CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT profiles_lng_range CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT profiles_coords_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE TABLE preferences (
  user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in   TEXT[] NOT NULL DEFAULT ARRAY['man', 'woman', 'non_binary'],
  min_age         SMALLINT NOT NULL DEFAULT 18,
  max_age         SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NULL DEFAULT 50,
  discoverable    BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance   BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT preferences_distance_range CHECK (max_distance_km IS NULL OR max_distance_km BETWEEN 1 AND 500),
  CONSTRAINT preferences_interested_valid CHECK (interested_in <@ ARRAY['man', 'woman', 'non_binary']::TEXT[])
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
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position     SMALLINT NOT NULL,
  storage_key  TEXT NOT NULL,
  thumb_key    TEXT NOT NULL,
  width        INTEGER NOT NULL,
  height       INTEGER NOT NULL,
  size_bytes   INTEGER NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_position_range CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT photos_user_position_unique UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_photos_user_created ON photos (user_id, created_at);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyages'), ('cooking', 'Cuisine'), ('music', 'Musique'), ('cinema', 'Cinéma'),
  ('reading', 'Lecture'), ('sport', 'Sport'), ('hiking', 'Randonnée'), ('art', 'Art'),
  ('photography', 'Photographie'), ('gaming', 'Jeux vidéo'), ('dancing', 'Danse'), ('yoga', 'Yoga'),
  ('animals', 'Animaux'), ('nature', 'Nature'), ('tech', 'Technologie'), ('theatre', 'Théâtre'),
  ('wine', 'Vin & gastronomie'), ('fitness', 'Fitness'), ('board_games', 'Jeux de société'), ('volunteering', 'Bénévolat');
