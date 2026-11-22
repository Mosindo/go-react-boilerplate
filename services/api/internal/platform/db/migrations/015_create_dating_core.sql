-- Dating domain: profiles, preferences, interests, photos, swipes, matches,
-- safety (blocks/reports) and account recovery.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Demo accounts created by cmd/seed are flagged so they can never be confused
-- with real members (they are badged in the UI and purgeable in one query).
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NULL,
  birthdate DATE NULL,
  gender TEXT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  -- Approximate position only: coordinates are rounded server-side to a
  -- ~1 km grid before storage and are never returned to other users.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  city TEXT NOT NULL DEFAULT '',
  location_updated_at TIMESTAMPTZ NULL,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  moderation_status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_gender CHECK (gender IS NULL OR gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_first_name CHECK (first_name IS NULL OR char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_job_title CHECK (char_length(job_title) <= 80),
  CONSTRAINT chk_profiles_city CHECK (char_length(city) <= 80),
  CONSTRAINT chk_profiles_latitude CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_longitude CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL)),
  CONSTRAINT chk_profiles_moderation CHECK (moderation_status IN ('active', 'under_review', 'suspended'))
);

CREATE INDEX IF NOT EXISTS idx_profiles_discovery_geo
  ON profiles (latitude, longitude)
  WHERE discoverable = TRUE AND moderation_status = 'active' AND latitude IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_birthdate
  ON profiles (birthdate);

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT '{}',
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]),
  CONSTRAINT chk_preferences_age CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  category TEXT NOT NULL
);

INSERT INTO interests (slug, label, category) VALUES
  ('hiking', 'Randonnée', 'outdoors'),
  ('cycling', 'Vélo', 'outdoors'),
  ('climbing', 'Escalade', 'outdoors'),
  ('surfing', 'Surf', 'outdoors'),
  ('camping', 'Camping', 'outdoors'),
  ('gardening', 'Jardinage', 'outdoors'),
  ('running', 'Course à pied', 'sport'),
  ('yoga', 'Yoga', 'sport'),
  ('swimming', 'Natation', 'sport'),
  ('football', 'Football', 'sport'),
  ('tennis', 'Tennis', 'sport'),
  ('dance', 'Danse', 'sport'),
  ('gym', 'Musculation', 'sport'),
  ('cooking', 'Cuisine', 'food'),
  ('baking', 'Pâtisserie', 'food'),
  ('wine', 'Vin', 'food'),
  ('coffee', 'Café', 'food'),
  ('vegetarian', 'Cuisine végé', 'food'),
  ('brunch', 'Brunch', 'food'),
  ('music', 'Musique', 'culture'),
  ('concerts', 'Concerts', 'culture'),
  ('cinema', 'Cinéma', 'culture'),
  ('series', 'Séries', 'culture'),
  ('reading', 'Lecture', 'culture'),
  ('theatre', 'Théâtre', 'culture'),
  ('museums', 'Musées', 'culture'),
  ('photography', 'Photographie', 'creative'),
  ('drawing', 'Dessin', 'creative'),
  ('writing', 'Écriture', 'creative'),
  ('diy', 'Bricolage', 'creative'),
  ('fashion', 'Mode', 'creative'),
  ('travel', 'Voyages', 'lifestyle'),
  ('pets', 'Animaux', 'lifestyle'),
  ('volunteering', 'Bénévolat', 'lifestyle'),
  ('meditation', 'Méditation', 'lifestyle'),
  ('board_games', 'Jeux de société', 'lifestyle'),
  ('video_games', 'Jeux vidéo', 'lifestyle'),
  ('tech', 'Tech', 'lifestyle'),
  ('languages', 'Langues', 'lifestyle'),
  ('nightlife', 'Sorties', 'lifestyle')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest
  ON user_interests (interest_id, user_id);

CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  mime_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  position SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT chk_photos_size CHECK (size_bytes > 0),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);

CREATE INDEX IF NOT EXISTS idx_photos_user_position
  ON photos (user_id, position);

-- One decision per (swiper, target): prevents repeated likes and re-showing
-- already processed profiles.
CREATE TABLE IF NOT EXISTS swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_id <> target_id)
);

CREATE INDEX IF NOT EXISTS idx_swipes_target_likes
  ON swipes (target_id, created_at DESC)
  WHERE action = 'like';

-- A match is stored once per pair with user_a_id < user_b_id.
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id UUID NULL REFERENCES conversations(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_matches_order CHECK (user_a_id < user_b_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_user_b
  ON matches (user_b_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_matches_user_a
  ON matches (user_a_id, created_at DESC);

CREATE TABLE IF NOT EXISTS blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT chk_blocks_not_self CHECK (blocker_id <> blocked_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocked
  ON blocks (blocked_id, blocker_id);

CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_reports_reason CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT chk_reports_details CHECK (char_length(details) <= 1000)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_open_pair
  ON reports (reporter_id, reported_id)
  WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_reports_reported_status
  ON reports (reported_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user
  ON password_reset_tokens (user_id, created_at DESC);

-- Chat: local "clear conversation" support.
ALTER TABLE conversation_participants
  ADD COLUMN IF NOT EXISTS cleared_at TIMESTAMPTZ NULL;

-- Great-circle distance in kilometres (haversine). PostGIS is intentionally
-- not required; discovery pre-filters with a bounding box on an index first.
CREATE OR REPLACE FUNCTION geo_distance_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
  SELECT 2 * 6371.0088 * asin(sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2) +
    cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  ))
$$;
