-- Core schema for the dating application.
-- Every table uses explicit constraints, foreign keys with deliberate ON DELETE
-- behaviour, and the indexes required by the hot query paths.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Great-circle distance in kilometres (haversine). Inputs are the rounded,
-- approximate coordinates stored on profiles.
CREATE OR REPLACE FUNCTION distance_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
  SELECT 2 * 6371 * asin(sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  ))
$$;

-- ---------------------------------------------------------------------------
-- Accounts & sessions
-- ---------------------------------------------------------------------------

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  is_demo BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_users_email_normalized CHECK (email = lower(btrim(email)) AND length(email) BETWEEN 3 AND 254)
);

CREATE UNIQUE INDEX idx_users_email ON users (email);

CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  user_agent TEXT NULL,
  ip_address INET NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ NULL,
  last_used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions (token_hash);
CREATE INDEX idx_sessions_user_created_at ON sessions (user_id, created_at DESC);

CREATE TABLE password_reset_codes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_codes_user_created_at ON password_reset_codes (user_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- Profiles, preferences, interests, photos
-- ---------------------------------------------------------------------------

CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL DEFAULT '',
  birthdate DATE NULL,
  gender TEXT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  height_cm SMALLINT NULL,
  city TEXT NOT NULL DEFAULT '',
  -- Coordinates are rounded server-side (~1 km grid) before storage. The exact
  -- device position is never persisted.
  approx_latitude DOUBLE PRECISION NULL,
  approx_longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  is_paused BOOLEAN NOT NULL DEFAULT FALSE,
  hide_distance BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_gender CHECK (gender IS NULL OR gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_first_name CHECK (length(first_name) <= 40),
  CONSTRAINT chk_profiles_bio CHECK (length(bio) <= 500),
  CONSTRAINT chk_profiles_job_title CHECK (length(job_title) <= 60),
  CONSTRAINT chk_profiles_city CHECK (length(city) <= 80),
  CONSTRAINT chk_profiles_height CHECK (height_cm IS NULL OR height_cm BETWEEN 120 AND 230),
  CONSTRAINT chk_profiles_latitude CHECK (approx_latitude IS NULL OR approx_latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_longitude CHECK (approx_longitude IS NULL OR approx_longitude BETWEEN -180 AND 180)
);

CREATE INDEX idx_profiles_discovery ON profiles (gender, birthdate) WHERE is_paused = FALSE;
CREATE INDEX idx_profiles_location ON profiles (approx_latitude, approx_longitude) WHERE approx_latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['woman', 'man', 'nonbinary']::TEXT[],
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  ),
  CONSTRAINT chk_preferences_age CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE interests (
  id SMALLINT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

CREATE TABLE user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX idx_user_interests_interest ON user_interests (interest_id, user_id);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  mime_type TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  position SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT chk_photos_mime CHECK (mime_type IN ('image/jpeg')),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_photos_user_position ON photos (user_id, position);

-- ---------------------------------------------------------------------------
-- Matching
-- ---------------------------------------------------------------------------

CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_id <> target_id)
);

CREATE INDEX idx_swipes_target_action ON swipes (target_id, action, created_at DESC);

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  -- user_a_id < user_b_id guarantees a single row per pair.
  CONSTRAINT chk_matches_order CHECK (user_a_id < user_b_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX idx_matches_user_b ON matches (user_b_id);

-- ---------------------------------------------------------------------------
-- Chat
-- ---------------------------------------------------------------------------

CREATE TABLE conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  last_message_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_participants (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_at TIMESTAMPTZ NULL,
  -- Local deletion: messages older than cleared_at are hidden for this user
  -- only, and the conversation is hidden until a new message arrives.
  cleared_at TIMESTAMPTZ NULL,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, conversation_id);

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  CONSTRAINT chk_messages_body CHECK (length(btrim(body)) BETWEEN 1 AND 2000)
);

CREATE INDEX idx_messages_conversation_created_at ON messages (conversation_id, created_at DESC, id DESC);

-- ---------------------------------------------------------------------------
-- Safety & moderation
-- ---------------------------------------------------------------------------

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT chk_blocks_not_self CHECK (blocker_id <> blocked_id)
);

CREATE INDEX idx_blocks_blocked ON blocks (blocked_id, blocker_id);

CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Reports survive account deletion so moderators keep the history.
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_reports_reason CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT chk_reports_details CHECK (length(details) <= 1000)
);

CREATE INDEX idx_reports_status_created_at ON reports (status, created_at DESC);
CREATE INDEX idx_reports_reported ON reports (reported_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- Notifications
-- ---------------------------------------------------------------------------

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  -- Used to collapse repeated notifications (e.g. several messages in one
  -- conversation) into a single unread row.
  group_key TEXT NULL,
  read_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_notifications_type CHECK (type IN ('match', 'message', 'like'))
);

CREATE INDEX idx_notifications_user_created_at ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_user_unread ON notifications (user_id) WHERE read_at IS NULL;
CREATE UNIQUE INDEX idx_notifications_unread_group ON notifications (user_id, group_key) WHERE read_at IS NULL AND group_key IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Reference data (not demo data): the catalogue of selectable interests.
-- ---------------------------------------------------------------------------

INSERT INTO interests (id, slug, label) VALUES
  (1, 'hiking', 'Randonnée'),
  (2, 'cooking', 'Cuisine'),
  (3, 'travel', 'Voyages'),
  (4, 'music', 'Musique'),
  (5, 'concerts', 'Concerts'),
  (6, 'cinema', 'Cinéma'),
  (7, 'reading', 'Lecture'),
  (8, 'photography', 'Photographie'),
  (9, 'yoga', 'Yoga'),
  (10, 'running', 'Course à pied'),
  (11, 'climbing', 'Escalade'),
  (12, 'cycling', 'Vélo'),
  (13, 'swimming', 'Natation'),
  (14, 'football', 'Football'),
  (15, 'gaming', 'Jeux vidéo'),
  (16, 'board_games', 'Jeux de société'),
  (17, 'art', 'Art'),
  (18, 'museums', 'Musées'),
  (19, 'theatre', 'Théâtre'),
  (20, 'dance', 'Danse'),
  (21, 'wine', 'Vin'),
  (22, 'coffee', 'Café'),
  (23, 'gardening', 'Jardinage'),
  (24, 'animals', 'Animaux'),
  (25, 'volunteering', 'Bénévolat'),
  (26, 'tech', 'Tech'),
  (27, 'science', 'Sciences'),
  (28, 'languages', 'Langues'),
  (29, 'meditation', 'Méditation'),
  (30, 'fashion', 'Mode'),
  (31, 'podcasts', 'Podcasts'),
  (32, 'writing', 'Écriture');
