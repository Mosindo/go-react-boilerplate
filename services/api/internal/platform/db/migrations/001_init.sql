-- Core schema for the dating application.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT users_email_lowercase CHECK (email = lower(email))
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
  code_hash  TEXT NOT NULL,
  attempts   SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at    TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_password_resets_user ON password_resets (user_id, created_at DESC);

-- Public profile. Coordinates are stored rounded (~1 km) on purpose: the exact
-- position of a user is never persisted.
CREATE TABLE profiles (
  user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name  TEXT NOT NULL,
  birth_date  DATE NOT NULL,
  gender      TEXT NOT NULL,
  bio         TEXT NOT NULL DEFAULT '',
  city        TEXT NOT NULL DEFAULT '',
  latitude    DOUBLE PRECISION NULL,
  longitude   DOUBLE PRECISION NULL,
  is_visible  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_gender_valid CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT profiles_first_name_len CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT profiles_bio_len CHECK (char_length(bio) <= 500),
  CONSTRAINT profiles_lat_range CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT profiles_lng_range CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT profiles_coords_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_discovery ON profiles (is_visible, gender, birth_date);
CREATE INDEX idx_profiles_geo ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in   TEXT[] NOT NULL DEFAULT '{}',
  min_age         SMALLINT NOT NULL DEFAULT 18,
  max_age         SMALLINT NOT NULL DEFAULT 99,
  max_distance_km INTEGER NOT NULL DEFAULT 50,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (min_age >= 18 AND max_age <= 120 AND min_age <= max_age),
  CONSTRAINT preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500),
  CONSTRAINT preferences_interested_valid CHECK (interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[])
);

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
  CONSTRAINT photos_position_range CHECK (position BETWEEN 0 AND 8),
  CONSTRAINT photos_user_position_unique UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);
CREATE UNIQUE INDEX idx_photos_storage_key ON photos (storage_key);
CREATE INDEX idx_photos_user ON photos (user_id, position);

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

-- One row per (actor, target): a user can only ever be judged once by another.
CREATE TABLE swipes (
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind         TEXT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (from_user_id, to_user_id),
  CONSTRAINT swipes_kind_valid CHECK (kind IN ('like', 'pass')),
  CONSTRAINT swipes_not_self CHECK (from_user_id <> to_user_id)
);
CREATE INDEX idx_swipes_incoming_likes ON swipes (to_user_id, created_at DESC) WHERE kind = 'like';

-- Canonical ordering (user_a < user_b) guarantees a single match per pair.
CREATE TABLE matches (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT matches_ordered CHECK (user_a_id < user_b_id),
  CONSTRAINT matches_pair_unique UNIQUE (user_a_id, user_b_id)
);
CREATE INDEX idx_matches_user_b ON matches (user_b_id);

CREATE TABLE conversations (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id        UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_participants (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_message_id BIGINT NOT NULL DEFAULT 0,
  hidden_at       TIMESTAMPTZ NULL,
  PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, conversation_id);

CREATE TABLE messages (
  id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body            TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT messages_body_len CHECK (char_length(body) BETWEEN 1 AND 2000)
);
CREATE INDEX idx_messages_conversation ON messages (conversation_id, created_at DESC, id DESC);

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT blocks_not_self CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id);

CREATE TABLE reports (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason      TEXT NOT NULL,
  details     TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT 'open',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT reports_reason_valid CHECK (reason IN ('spam', 'fake', 'harassment', 'inappropriate', 'underage', 'other')),
  CONSTRAINT reports_status_valid CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT reports_not_self CHECK (reporter_id <> reported_id),
  CONSTRAINT reports_details_len CHECK (char_length(details) <= 1000)
);
CREATE INDEX idx_reports_status ON reports (status, created_at DESC);
CREATE INDEX idx_reports_reported ON reports (reported_id);

CREATE TABLE notifications (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type       TEXT NOT NULL,
  title      TEXT NOT NULL,
  body       TEXT NOT NULL DEFAULT '',
  data       JSONB NOT NULL DEFAULT '{}'::jsonb,
  read_at    TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_user ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_unread ON notifications (user_id) WHERE read_at IS NULL;

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyages'), ('cooking', 'Cuisine'), ('music', 'Musique'), ('cinema', 'Cinéma'),
  ('reading', 'Lecture'), ('sport', 'Sport'), ('hiking', 'Randonnée'), ('photography', 'Photo'),
  ('gaming', 'Jeux vidéo'), ('art', 'Art'), ('dance', 'Danse'), ('yoga', 'Yoga'),
  ('animals', 'Animaux'), ('nature', 'Nature'), ('technology', 'Tech'), ('fashion', 'Mode'),
  ('theatre', 'Théâtre'), ('wine', 'Gastronomie'), ('cycling', 'Vélo'), ('volunteering', 'Bénévolat');
