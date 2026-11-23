-- Dating product core schema.
-- Removes boilerplate modules that do not belong to a free dating product
-- (billing, public posts/comments/votes, metadata-only files) and introduces
-- profiles, preferences, photos, interests, swipes, matches, blocks, reports
-- and password recovery.

DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS subscriptions;

-- Legacy boilerplate chats were not scoped to mutual matches. They cannot be
-- migrated safely, so they are removed. Messages cascade with conversations.
DELETE FROM conversations WHERE kind <> 'match';

ALTER TABLE users DROP COLUMN IF EXISTS display_name;
ALTER TABLE users DROP COLUMN IF EXISTS bio;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ NULL;

CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NULL CHECK (first_name IS NULL OR char_length(first_name) BETWEEN 1 AND 40),
  birthdate DATE NULL,
  gender TEXT NULL CHECK (gender IN ('woman', 'man', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NULL CHECK (city IS NULL OR char_length(city) <= 80),
  -- Coordinates are snapped server-side to a ~1 km grid before storage.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  location_updated_at TIMESTAMPTZ NULL,
  is_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  hide_distance BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_profiles_discovery
  ON profiles (last_active_at DESC, user_id)
  WHERE is_discoverable = TRUE AND latitude IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_lat_lon
  ON profiles (latitude, longitude)
  WHERE latitude IS NOT NULL;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT '{}',
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_genders CHECK (interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]),
  CONSTRAINT chk_preferences_age CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 300)
);

INSERT INTO profiles (user_id) SELECT id FROM users ON CONFLICT (user_id) DO NOTHING;
INSERT INTO preferences (user_id) SELECT id FROM users ON CONFLICT (user_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  storage_key TEXT NOT NULL UNIQUE,
  mime_type TEXT NOT NULL CHECK (mime_type IN ('image/jpeg')),
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  width INTEGER NOT NULL CHECK (width > 0),
  height INTEGER NOT NULL CHECK (height > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  -- Deferred so photos can be reordered inside a single transaction.
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO interests (slug, label) VALUES
  ('hiking', 'Randonnée'),
  ('running', 'Course à pied'),
  ('yoga', 'Yoga'),
  ('climbing', 'Escalade'),
  ('cycling', 'Vélo'),
  ('swimming', 'Natation'),
  ('football', 'Football'),
  ('tennis', 'Tennis'),
  ('dance', 'Danse'),
  ('cooking', 'Cuisine'),
  ('baking', 'Pâtisserie'),
  ('wine', 'Vin'),
  ('coffee', 'Café'),
  ('restaurants', 'Restaurants'),
  ('travel', 'Voyages'),
  ('road-trips', 'Road trips'),
  ('camping', 'Camping'),
  ('photography', 'Photographie'),
  ('painting', 'Peinture'),
  ('museums', 'Musées'),
  ('theatre', 'Théâtre'),
  ('cinema', 'Cinéma'),
  ('series', 'Séries'),
  ('reading', 'Lecture'),
  ('writing', 'Écriture'),
  ('poetry', 'Poésie'),
  ('music', 'Musique'),
  ('concerts', 'Concerts'),
  ('festivals', 'Festivals'),
  ('jazz', 'Jazz'),
  ('electro', 'Électro'),
  ('guitar', 'Guitare'),
  ('piano', 'Piano'),
  ('video-games', 'Jeux vidéo'),
  ('board-games', 'Jeux de société'),
  ('tech', 'Tech'),
  ('science', 'Sciences'),
  ('languages', 'Langues'),
  ('volunteering', 'Bénévolat'),
  ('ecology', 'Écologie'),
  ('gardening', 'Jardinage'),
  ('animals', 'Animaux'),
  ('dogs', 'Chiens'),
  ('cats', 'Chats'),
  ('meditation', 'Méditation'),
  ('fashion', 'Mode'),
  ('design', 'Design'),
  ('architecture', 'Architecture')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);

CREATE TABLE IF NOT EXISTS swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_id <> target_id)
);

CREATE INDEX IF NOT EXISTS idx_swipes_target_action
  ON swipes (target_id, action, swiper_id);

CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id UUID NOT NULL UNIQUE REFERENCES conversations(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_matches_ordered_pair CHECK (user_a_id < user_b_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_user_b
  ON matches (user_b_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_matches_user_a_created
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
  -- Reports survive the reporter's account deletion for moderation history.
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'scam', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_reports_not_self CHECK (reporter_id IS NULL OR reporter_id <> reported_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_open_pair
  ON reports (reporter_id, reported_id)
  WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_reports_reported_created
  ON reports (reported_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reports_status_created
  ON reports (status, created_at DESC);

CREATE TABLE IF NOT EXISTS password_resets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_resets_user_created
  ON password_resets (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_conversation_participants_user_conversation
  ON conversation_participants (user_id, conversation_id);

-- Great-circle distance in kilometers (haversine).
CREATE OR REPLACE FUNCTION km_between(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
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
