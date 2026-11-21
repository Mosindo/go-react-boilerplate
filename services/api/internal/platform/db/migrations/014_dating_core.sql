-- Dating core schema: profiles, preferences, photos, interests, swipes,
-- matches, blocks, reports and password recovery codes.

CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  height_cm SMALLINT NULL,
  relationship_goal TEXT NULL,
  -- Coordinates are coarsened server-side (~1 km grid) before being stored.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  -- Set automatically when several distinct members report a profile; hidden
  -- from discovery until a moderator reviews it.
  is_suspended BOOLEAN NOT NULL DEFAULT FALSE,
  -- Demo accounts are created only by the seed command and never in production.
  is_demo BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_first_name CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_city CHECK (char_length(city) <= 80),
  CONSTRAINT chk_profiles_job_title CHECK (char_length(job_title) <= 80),
  CONSTRAINT chk_profiles_height CHECK (height_cm IS NULL OR height_cm BETWEEN 120 AND 230),
  CONSTRAINT chk_profiles_goal CHECK (
    relationship_goal IS NULL
    OR relationship_goal IN ('long_term', 'short_term', 'friendship', 'undecided')
  ),
  CONSTRAINT chk_profiles_latitude CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_longitude CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_profiles_discovery
  ON profiles (gender, birthdate)
  WHERE is_visible AND NOT is_suspended;

CREATE INDEX IF NOT EXISTS idx_profiles_location
  ON profiles (latitude, longitude)
  WHERE latitude IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_demo
  ON profiles (user_id)
  WHERE is_demo;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['woman', 'man', 'nonbinary'],
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (
    cardinality(interested_in) >= 1
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']
  ),
  CONSTRAINT chk_preferences_ages CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  thumb_storage_key TEXT NOT NULL UNIQUE,
  position SMALLINT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT chk_photos_dimensions CHECK (width > 0 AND height > 0 AND size_bytes > 0),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLINT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO interests (id, slug, label) VALUES
  (1, 'hiking', 'Randonnée'),
  (2, 'cooking', 'Cuisine'),
  (3, 'travel', 'Voyages'),
  (4, 'music', 'Musique'),
  (5, 'concerts', 'Concerts'),
  (6, 'cinema', 'Cinéma'),
  (7, 'series', 'Séries'),
  (8, 'reading', 'Lecture'),
  (9, 'photography', 'Photographie'),
  (10, 'art', 'Art & expos'),
  (11, 'theatre', 'Théâtre'),
  (12, 'dance', 'Danse'),
  (13, 'running', 'Course à pied'),
  (14, 'cycling', 'Vélo'),
  (15, 'yoga', 'Yoga'),
  (16, 'fitness', 'Sport en salle'),
  (17, 'swimming', 'Natation'),
  (18, 'climbing', 'Escalade'),
  (19, 'football', 'Football'),
  (20, 'tennis', 'Tennis'),
  (21, 'gaming', 'Jeux vidéo'),
  (22, 'board_games', 'Jeux de société'),
  (23, 'tech', 'Tech'),
  (24, 'science', 'Sciences'),
  (25, 'nature', 'Nature'),
  (26, 'gardening', 'Jardinage'),
  (27, 'animals', 'Animaux'),
  (28, 'wine', 'Vin'),
  (29, 'coffee', 'Café'),
  (30, 'food', 'Bonnes tables'),
  (31, 'volunteering', 'Bénévolat'),
  (32, 'languages', 'Langues'),
  (33, 'writing', 'Écriture'),
  (34, 'fashion', 'Mode'),
  (35, 'meditation', 'Méditation'),
  (36, 'beach', 'Plage'),
  (37, 'mountains', 'Montagne'),
  (38, 'skiing', 'Ski'),
  (39, 'podcasts', 'Podcasts'),
  (40, 'diy', 'Bricolage')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest
  ON user_interests (interest_id, user_id);

-- One row per (actor, target): a profile can only be liked or passed once,
-- which also guarantees processed profiles never resurface in discovery.
CREATE TABLE IF NOT EXISTS swipes (
  actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (actor_user_id, target_user_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (actor_user_id <> target_user_id)
);

CREATE INDEX IF NOT EXISTS idx_swipes_target_action
  ON swipes (target_user_id, action, actor_user_id);

-- A match is stored once per pair with an ordered (low, high) key so that it
-- can never be duplicated regardless of which side liked last.
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_low_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_high_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_matches_order CHECK (user_low_id < user_high_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_low_id, user_high_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_high_created_at
  ON matches (user_high_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_matches_low_created_at
  ON matches (user_low_id, created_at DESC);

ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS match_id UUID NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_conversations_match_id') THEN
    ALTER TABLE conversations
      ADD CONSTRAINT fk_conversations_match_id
      FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE;
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_match_id
  ON conversations (match_id)
  WHERE match_id IS NOT NULL;

-- Per-participant "clear conversation" marker (local deletion).
ALTER TABLE conversation_participants
  ADD COLUMN IF NOT EXISTS cleared_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_conversation_participants_conversation_user
  ON conversation_participants (conversation_id, user_id);

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
  -- Snapshot of the most recent messages exchanged, kept for moderators even
  -- after the block removes the conversation.
  evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_reports_reason CHECK (
    reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')
  ),
  CONSTRAINT chk_reports_details CHECK (char_length(details) <= 1000),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'actioned', 'dismissed'))
);

CREATE INDEX IF NOT EXISTS idx_reports_status_created_at
  ON reports (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reports_reported_created_at
  ON reports (reported_user_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_open_pair
  ON reports (reporter_user_id, reported_user_id)
  WHERE status = 'open';

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

CREATE INDEX IF NOT EXISTS idx_notifications_user_type_unread
  ON notifications (user_id, type)
  WHERE is_read = FALSE;

-- Great-circle distance used by discovery and profile cards.
CREATE OR REPLACE FUNCTION haversine_km(lat1 DOUBLE PRECISION, lng1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lng2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
AS $$
  SELECT 2 * 6371 * asin(least(1, sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2)
    + cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lng2 - lng1) / 2), 2)
  )))
$$;
