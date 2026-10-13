-- Dating core schema.
-- The product pivot replaces the legacy direct-message and notification tables
-- (kept only by earlier migrations) with match-based conversations.

DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS conversation_participants CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;

ALTER TABLE users ALTER COLUMN organization_id DROP NOT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE users ADD CONSTRAINT users_status_check CHECK (status IN ('active', 'suspended'));

CREATE TABLE password_resets (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts INT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('woman', 'man', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NULL CHECK (city IS NULL OR char_length(city) <= 80),
  -- Coarse coordinates only (rounded to 2 decimals, roughly 1 km) by the service.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_latitude ON profiles (latitude) WHERE latitude IS NOT NULL;
CREATE INDEX idx_profiles_gender_birth_date ON profiles (gender, birth_date);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  ),
  min_age INT NOT NULL DEFAULT 18 CHECK (min_age >= 18),
  max_age INT NOT NULL DEFAULT 99 CHECK (max_age <= 120),
  -- NULL means "anywhere".
  max_distance_km INT NULL CHECK (max_distance_km IS NULL OR max_distance_km BETWEEN 1 AND 20000),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (min_age <= max_age)
);

CREATE TABLE interests (
  id SMALLINT PRIMARY KEY,
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
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  content_type TEXT NOT NULL DEFAULT 'image/jpeg',
  width INT NOT NULL,
  height INT NOT NULL,
  data BYTEA NOT NULL,
  thumbnail BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_photos_user_position ON photos (user_id, position);

CREATE TABLE likes (
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (from_user_id, to_user_id),
  CHECK (from_user_id <> to_user_id)
);
CREATE INDEX idx_likes_to_user_action ON likes (to_user_id, action);

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unmatched_at TIMESTAMPTZ NULL,
  unmatched_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  -- Canonical ordering guarantees a single row per pair.
  CHECK (user_a < user_b),
  UNIQUE (user_a, user_b)
);
CREATE INDEX idx_matches_user_b ON matches (user_b);

CREATE TABLE conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  last_message_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_participants (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  -- Local deletion: hidden from the participant's list until a new message arrives.
  hidden_at TIMESTAMPTZ NULL,
  PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, conversation_id);

CREATE TABLE messages (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_messages_conversation_created_at ON messages (conversation_id, created_at, id);

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id);

CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL CHECK (reason IN ('spam', 'fake', 'harassment', 'inappropriate', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (reporter_id <> reported_id)
);
CREATE INDEX idx_reports_status_created ON reports (status, created_at);
CREATE INDEX idx_reports_reported ON reports (reported_id);

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL CHECK (type IN ('match', 'message')),
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  read_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);
-- One unread "message" notification per conversation (aggregated, not spammy).
CREATE UNIQUE INDEX idx_notifications_unread_message
  ON notifications (user_id, (data->>'conversationId'))
  WHERE type = 'message' AND read_at IS NULL;

CREATE FUNCTION haversine_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE SQL IMMUTABLE PARALLEL SAFE AS $$
  SELECT 12742 * asin(sqrt(least(1.0,
    power(sin(radians(lat2 - lat1) / 2), 2)
    + cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  )))
$$;

-- Reference data (not demo data): interests users can pick from.
INSERT INTO interests (id, slug, label) VALUES
  (1, 'travel', 'Voyages'), (2, 'cooking', 'Cuisine'), (3, 'music', 'Musique'),
  (4, 'cinema', 'Cinéma'), (5, 'reading', 'Lecture'), (6, 'sport', 'Sport'),
  (7, 'hiking', 'Randonnée'), (8, 'art', 'Art'), (9, 'photography', 'Photo'),
  (10, 'gaming', 'Jeux vidéo'), (11, 'dancing', 'Danse'), (12, 'yoga', 'Yoga'),
  (13, 'coffee', 'Café'), (14, 'wine', 'Vin'), (15, 'animals', 'Animaux'),
  (16, 'nature', 'Nature'), (17, 'tech', 'Tech'), (18, 'theatre', 'Théâtre'),
  (19, 'festivals', 'Festivals'), (20, 'cycling', 'Vélo'), (21, 'running', 'Course à pied'),
  (22, 'board-games', 'Jeux de société'), (23, 'volunteering', 'Bénévolat'), (24, 'languages', 'Langues');
