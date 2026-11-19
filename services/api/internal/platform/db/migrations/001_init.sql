-- Core schema for the dating application.
-- Migrations are applied once each and tracked in schema_migrations (see db.RunMigrations).

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------------------
-- Accounts and sessions
-- ---------------------------------------------------------------------------
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  is_demo BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT users_email_lowercase CHECK (email = lower(btrim(email))),
  CONSTRAINT users_email_length CHECK (char_length(email) BETWEEN 3 AND 254)
);
CREATE UNIQUE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_demo ON users (is_demo) WHERE is_demo;

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

CREATE TABLE password_resets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_password_resets_user_created_at ON password_resets (user_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- Profiles, preferences, interests, photos
-- ---------------------------------------------------------------------------
CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  looking_for TEXT NOT NULL DEFAULT 'unsure',
  city TEXT NOT NULL DEFAULT '',
  -- Coordinates are rounded server-side (~1 km grid) before being stored.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_first_name_length CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT profiles_gender_valid CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT profiles_bio_length CHECK (char_length(bio) <= 500),
  CONSTRAINT profiles_job_title_length CHECK (char_length(job_title) <= 60),
  CONSTRAINT profiles_city_length CHECK (char_length(city) <= 80),
  CONSTRAINT profiles_looking_for_valid CHECK (looking_for IN ('relationship', 'casual', 'friendship', 'unsure')),
  CONSTRAINT profiles_latitude_range CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT profiles_longitude_range CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_gender_birthdate ON profiles (gender, birthdate);
CREATE INDEX idx_profiles_lat_lng ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;
CREATE INDEX idx_profiles_last_active ON profiles (last_active_at DESC);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  age_min SMALLINT NOT NULL DEFAULT 18,
  age_max SMALLINT NOT NULL DEFAULT 99,
  max_distance_km INTEGER NOT NULL DEFAULT 50,
  is_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_interested_in_valid CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  ),
  CONSTRAINT preferences_age_range CHECK (age_min >= 18 AND age_max <= 99 AND age_min <= age_max),
  CONSTRAINT preferences_distance_range CHECK (max_distance_km BETWEEN 1 AND 500)
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

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  position SMALLINT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_position_range CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT photos_user_position_unique UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- ---------------------------------------------------------------------------
-- Matching
-- ---------------------------------------------------------------------------
CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  decision TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT swipes_decision_valid CHECK (decision IN ('like', 'pass')),
  CONSTRAINT swipes_not_self CHECK (swiper_id <> target_id)
);
CREATE INDEX idx_swipes_target_decision ON swipes (target_id, decision, created_at DESC);

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unmatched_at TIMESTAMPTZ NULL,
  unmatched_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT matches_ordered_pair CHECK (user_a_id < user_b_id),
  CONSTRAINT matches_pair_unique UNIQUE (user_a_id, user_b_id)
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
  -- Local deletion: messages created before cleared_at are hidden for this participant.
  cleared_at TIMESTAMPTZ NULL,
  PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, conversation_id);

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  CONSTRAINT messages_body_length CHECK (char_length(body) BETWEEN 1 AND 2000)
);
CREATE INDEX idx_messages_conversation_created_at ON messages (conversation_id, created_at DESC, id DESC);

-- ---------------------------------------------------------------------------
-- Safety and moderation
-- ---------------------------------------------------------------------------
CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT blocks_not_self CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id, blocker_id);

CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Reports are kept for moderation even after either account is deleted.
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT reports_reason_valid CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')),
  CONSTRAINT reports_status_valid CHECK (status IN ('open', 'reviewed', 'actioned', 'dismissed')),
  CONSTRAINT reports_details_length CHECK (char_length(details) <= 1000)
);
CREATE INDEX idx_reports_reported_created_at ON reports (reported_id, created_at DESC);
CREATE INDEX idx_reports_status_created_at ON reports (status, created_at DESC);

-- ---------------------------------------------------------------------------
-- Notifications
-- ---------------------------------------------------------------------------
CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  actor_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id UUID NULL REFERENCES conversations(id) ON DELETE CASCADE,
  read_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  CONSTRAINT notifications_type_valid CHECK (type IN ('match', 'message'))
);
CREATE INDEX idx_notifications_user_created_at ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_user_unread ON notifications (user_id) WHERE read_at IS NULL;

-- ---------------------------------------------------------------------------
-- Reference data
-- ---------------------------------------------------------------------------
INSERT INTO interests (slug, label) VALUES
  ('hiking', 'Randonnée'),
  ('cooking', 'Cuisine'),
  ('travel', 'Voyages'),
  ('music', 'Musique'),
  ('concerts', 'Concerts'),
  ('cinema', 'Cinéma'),
  ('series', 'Séries'),
  ('reading', 'Lecture'),
  ('writing', 'Écriture'),
  ('photography', 'Photographie'),
  ('art', 'Art'),
  ('museums', 'Musées'),
  ('theatre', 'Théâtre'),
  ('dancing', 'Danse'),
  ('yoga', 'Yoga'),
  ('running', 'Course à pied'),
  ('cycling', 'Vélo'),
  ('climbing', 'Escalade'),
  ('swimming', 'Natation'),
  ('football', 'Football'),
  ('fitness', 'Fitness'),
  ('gaming', 'Jeux vidéo'),
  ('board_games', 'Jeux de société'),
  ('tech', 'Tech'),
  ('science', 'Sciences'),
  ('nature', 'Nature'),
  ('gardening', 'Jardinage'),
  ('animals', 'Animaux'),
  ('wine', 'Vin'),
  ('coffee', 'Café'),
  ('food', 'Gastronomie'),
  ('volunteering', 'Bénévolat'),
  ('languages', 'Langues'),
  ('meditation', 'Méditation'),
  ('fashion', 'Mode'),
  ('diy', 'Bricolage')
ON CONFLICT (slug) DO NOTHING;
