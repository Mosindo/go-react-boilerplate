-- Dating product schema: profiles, preferences, photos, interests, swipes,
-- matches, match-scoped conversations, blocks, reports and password resets.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  city TEXT NULL,
  -- Coordinates are snapped to a ~1 km grid by the API before storage.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  moderation_hidden_at TIMESTAMPTZ NULL,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_first_name CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_city CHECK (city IS NULL OR char_length(city) <= 80),
  CONSTRAINT chk_profiles_latitude CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_longitude CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_profiles_discoverable
  ON profiles (gender, birthdate)
  WHERE is_visible = TRUE AND moderation_hidden_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_location
  ON profiles (latitude, longitude)
  WHERE latitude IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_last_active
  ON profiles (last_active_at DESC);

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['woman', 'man', 'nonbinary'],
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km INTEGER NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']
  ),
  CONSTRAINT chk_preferences_age CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL,
  storage_key TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT chk_photos_dimensions CHECK (width > 0 AND height > 0 AND size_bytes > 0),
  CONSTRAINT uq_photos_storage_key UNIQUE (storage_key),
  -- Deferred so that a reorder can permute positions inside one transaction.
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Reference catalog (product data, not demo data).
INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyages'),
  ('hiking', 'Randonnée'),
  ('cooking', 'Cuisine'),
  ('wine', 'Vin'),
  ('coffee', 'Café'),
  ('reading', 'Lecture'),
  ('writing', 'Écriture'),
  ('cinema', 'Cinéma'),
  ('series', 'Séries'),
  ('music', 'Musique'),
  ('concerts', 'Concerts'),
  ('dancing', 'Danse'),
  ('art', 'Art'),
  ('photography', 'Photographie'),
  ('museums', 'Musées'),
  ('theatre', 'Théâtre'),
  ('running', 'Course à pied'),
  ('cycling', 'Vélo'),
  ('yoga', 'Yoga'),
  ('climbing', 'Escalade'),
  ('swimming', 'Natation'),
  ('football', 'Football'),
  ('tennis', 'Tennis'),
  ('gym', 'Musculation'),
  ('video_games', 'Jeux vidéo'),
  ('board_games', 'Jeux de société'),
  ('tech', 'Tech'),
  ('nature', 'Nature'),
  ('animals', 'Animaux'),
  ('gardening', 'Jardinage'),
  ('volunteering', 'Bénévolat'),
  ('meditation', 'Méditation'),
  ('fashion', 'Mode'),
  ('languages', 'Langues'),
  ('podcasts', 'Podcasts'),
  ('food', 'Bonnes tables')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest
  ON user_interests (interest_id, user_id);

-- A swipe is a like or a pass. One row per (swiper, target): a profile that
-- has been handled is never shown again and a like cannot be repeated.
CREATE TABLE IF NOT EXISTS swipes (
  swiper_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_user_id, target_user_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_user_id <> target_user_id)
);

CREATE INDEX IF NOT EXISTS idx_swipes_target_likes
  ON swipes (target_user_id, swiper_user_id)
  WHERE action = 'like';

-- A match is stored once per pair with ordered ids so duplicates are impossible.
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_low_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_high_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_matches_ordered CHECK (user_low_id < user_high_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_low_id, user_high_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_user_high
  ON matches (user_high_id, user_low_id);

ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS match_id UUID NULL REFERENCES matches(id) ON DELETE CASCADE;

ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_conversations_match_id
  ON conversations (match_id)
  WHERE match_id IS NOT NULL;

ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS client_id TEXT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_messages_client_id
  ON messages (conversation_id, sender_user_id, client_id)
  WHERE client_id IS NOT NULL;

ALTER TABLE messages
  ADD CONSTRAINT chk_messages_content_length
  CHECK (char_length(content) BETWEEN 1 AND 2000) NOT VALID;

CREATE TABLE IF NOT EXISTS blocks (
  blocker_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_user_id, blocked_user_id),
  CONSTRAINT chk_blocks_not_self CHECK (blocker_user_id <> blocked_user_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocked
  ON blocks (blocked_user_id, blocker_user_id);

CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  -- Snapshot of the conversation at report time so moderation still has
  -- context after an unmatch or a block deletes the conversation.
  evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_reports_reason CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')),
  CONSTRAINT chk_reports_details CHECK (char_length(details) <= 1000),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'dismissed'))
);

CREATE INDEX IF NOT EXISTS idx_reports_reported_created_at
  ON reports (reported_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reports_status_created_at
  ON reports (status, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS uq_reports_open_pair
  ON reports (reporter_user_id, reported_user_id)
  WHERE status = 'open';

CREATE TABLE IF NOT EXISTS password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_created_at
  ON password_reset_tokens (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sessions_user_active
  ON sessions (user_id)
  WHERE revoked_at IS NULL;
