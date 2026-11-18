-- Demo accounts created by cmd/seed are flagged so they can never be confused
-- with real members and can be purged in one statement.
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_users_is_demo ON users (is_demo) WHERE is_demo;

-- One profile per user. Coordinates are stored already rounded (about 1 km)
-- so the exact position of a member never reaches the database.
CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  height_cm SMALLINT NULL,
  relationship_goal TEXT NOT NULL DEFAULT 'unsure',
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  is_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  moderation_hidden BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_first_name CHECK (char_length(first_name) BETWEEN 1 AND 30),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_job_title CHECK (char_length(job_title) <= 60),
  CONSTRAINT chk_profiles_city CHECK (char_length(city) <= 60),
  CONSTRAINT chk_profiles_height CHECK (height_cm IS NULL OR height_cm BETWEEN 120 AND 230),
  CONSTRAINT chk_profiles_goal CHECK (relationship_goal IN ('long_term', 'short_term', 'friendship', 'unsure')),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL)),
  CONSTRAINT chk_profiles_latitude CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_longitude CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180)
);

CREATE INDEX IF NOT EXISTS idx_profiles_discovery
  ON profiles (gender, birthdate)
  WHERE is_discoverable AND NOT moderation_hidden;

CREATE INDEX IF NOT EXISTS idx_profiles_location
  ON profiles (latitude, longitude)
  WHERE latitude IS NOT NULL;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_ages CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500),
  CONSTRAINT chk_preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  )
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
  CONSTRAINT uq_photos_storage_key UNIQUE (storage_key),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  category TEXT NOT NULL
);

INSERT INTO interests (slug, label, category) VALUES
  ('hiking', 'Randonnée', 'Plein air'),
  ('climbing', 'Escalade', 'Plein air'),
  ('cycling', 'Vélo', 'Plein air'),
  ('running', 'Course à pied', 'Sport'),
  ('yoga', 'Yoga', 'Sport'),
  ('swimming', 'Natation', 'Sport'),
  ('football', 'Football', 'Sport'),
  ('dance', 'Danse', 'Sport'),
  ('cooking', 'Cuisine', 'Art de vivre'),
  ('wine', 'Vin', 'Art de vivre'),
  ('coffee', 'Café', 'Art de vivre'),
  ('brunch', 'Brunch', 'Art de vivre'),
  ('travel', 'Voyages', 'Art de vivre'),
  ('gardening', 'Jardinage', 'Art de vivre'),
  ('reading', 'Lecture', 'Culture'),
  ('cinema', 'Cinéma', 'Culture'),
  ('theatre', 'Théâtre', 'Culture'),
  ('museums', 'Musées', 'Culture'),
  ('photography', 'Photographie', 'Culture'),
  ('writing', 'Écriture', 'Culture'),
  ('concerts', 'Concerts', 'Musique'),
  ('jazz', 'Jazz', 'Musique'),
  ('electro', 'Électro', 'Musique'),
  ('playing_music', 'Jouer de la musique', 'Musique'),
  ('video_games', 'Jeux vidéo', 'Jeux'),
  ('board_games', 'Jeux de société', 'Jeux'),
  ('tech', 'Tech', 'Curiosité'),
  ('science', 'Sciences', 'Curiosité'),
  ('volunteering', 'Bénévolat', 'Engagement'),
  ('animals', 'Animaux', 'Engagement'),
  ('ecology', 'Écologie', 'Engagement'),
  ('meditation', 'Méditation', 'Bien-être')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest
  ON user_interests (interest_id, user_id);

-- A swipe is a single decision per (actor, target) pair: it can never be repeated.
CREATE TABLE IF NOT EXISTS swipes (
  actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  decision TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (actor_user_id, target_user_id),
  CONSTRAINT chk_swipes_decision CHECK (decision IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (actor_user_id <> target_user_id)
);

CREATE INDEX IF NOT EXISTS idx_swipes_target_likes
  ON swipes (target_user_id, actor_user_id)
  WHERE decision = 'like';

-- Matches are stored once per pair with an ordered (low, high) key.
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_low_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_high_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_matches_order CHECK (user_low_id < user_high_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_low_id, user_high_id),
  CONSTRAINT uq_matches_conversation UNIQUE (conversation_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_low_created_at ON matches (user_low_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_matches_high_created_at ON matches (user_high_id, created_at DESC);

CREATE TABLE IF NOT EXISTS blocks (
  blocker_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_user_id, blocked_user_id),
  CONSTRAINT chk_blocks_not_self CHECK (blocker_user_id <> blocked_user_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocked ON blocks (blocked_user_id, blocker_user_id);

-- Reports survive account deletion (SET NULL) so moderation history is kept.
CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_reports_reason CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')),
  CONSTRAINT chk_reports_details CHECK (char_length(details) <= 1000),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'dismissed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_reports_open_pair
  ON reports (reporter_user_id, reported_user_id)
  WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_reports_reported_created_at ON reports (reported_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_status_created_at ON reports (status, created_at DESC);

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

CREATE INDEX IF NOT EXISTS idx_sessions_active_user
  ON sessions (user_id)
  WHERE revoked_at IS NULL;
