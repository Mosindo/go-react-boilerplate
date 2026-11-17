-- Core schema for the dating application.
-- Every statement is idempotent so a partially applied migration can be retried safely.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------------------
-- Accounts and sessions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  -- Demo accounts are created only by the seed command and are flagged in every API payload.
  is_demo BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_users_email_normalized CHECK (email = lower(btrim(email)) AND length(email) BETWEEN 3 AND 254)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

CREATE TABLE IF NOT EXISTS sessions (
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions (token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user_created_at ON sessions (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS password_reset_codes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_codes_user_created_at
  ON password_reset_codes (user_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- Profiles, preferences and interests
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL DEFAULT '',
  birthdate DATE NULL,
  gender TEXT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  -- Coordinates are rounded to 2 decimals (~1 km) before storage; exact positions are never persisted.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_gender CHECK (gender IS NULL OR gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_first_name CHECK (length(first_name) <= 40),
  CONSTRAINT chk_profiles_bio CHECK (length(bio) <= 500),
  CONSTRAINT chk_profiles_job_title CHECK (length(job_title) <= 80),
  CONSTRAINT chk_profiles_city CHECK (length(city) <= 80),
  CONSTRAINT chk_profiles_latitude CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_longitude CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_profiles_discovery
  ON profiles (gender, birthdate)
  WHERE discoverable = TRUE AND latitude IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_lat_lng
  ON profiles (latitude, longitude)
  WHERE latitude IS NOT NULL;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['woman', 'man', 'nonbinary']::TEXT[],
  age_min SMALLINT NOT NULL DEFAULT 18,
  age_max SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  ),
  CONSTRAINT chk_preferences_age CHECK (age_min >= 18 AND age_max <= 99 AND age_min <= age_max),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  category TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest ON user_interests (interest_id, user_id);

-- ---------------------------------------------------------------------------
-- Photos
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL,
  position SMALLINT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  mime_type TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT chk_photos_mime CHECK (mime_type IN ('image/jpeg'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_photos_storage_key ON photos (storage_key);
CREATE INDEX IF NOT EXISTS idx_photos_user_position ON photos (user_id, position);

-- ---------------------------------------------------------------------------
-- Matching
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_id <> target_id)
);

CREATE INDEX IF NOT EXISTS idx_swipes_target_swiper ON swipes (target_id, swiper_id, action);

CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ NULL,
  ended_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  -- Users are stored in a canonical order so a pair can only ever have one match row.
  CONSTRAINT chk_matches_order CHECK (user_a_id < user_b_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_matches_pair ON matches (user_a_id, user_b_id);
CREATE INDEX IF NOT EXISTS idx_matches_user_b ON matches (user_b_id, user_a_id);

-- ---------------------------------------------------------------------------
-- Chat
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  last_message_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_match ON conversations (match_id);

CREATE TABLE IF NOT EXISTS conversation_participants (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_at TIMESTAMPTZ NULL,
  -- Local deletion: messages older than hidden_at are hidden for this participant only.
  hidden_at TIMESTAMPTZ NULL,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_participants_user
  ON conversation_participants (user_id, conversation_id);

CREATE TABLE IF NOT EXISTS messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  CONSTRAINT chk_messages_content CHECK (length(content) BETWEEN 1 AND 2000)
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_created_at
  ON messages (conversation_id, created_at DESC, id DESC);

-- ---------------------------------------------------------------------------
-- Safety
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT chk_blocks_not_self CHECK (blocker_id <> blocked_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocked ON blocks (blocked_id, blocker_id);

CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Reports outlive account deletion so moderators keep the history.
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_reports_reason CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'scam', 'underage', 'other')),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT chk_reports_details CHECK (length(details) <= 1000)
);

CREATE INDEX IF NOT EXISTS idx_reports_status_created_at ON reports (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_open_pair
  ON reports (reporter_id, reported_id)
  WHERE status = 'open';

-- ---------------------------------------------------------------------------
-- Notifications
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  is_read BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created_at
  ON notifications (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
  ON notifications (user_id)
  WHERE is_read = FALSE;
