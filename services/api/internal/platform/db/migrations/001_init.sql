CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------- accounts
CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT NOT NULL UNIQUE CHECK (email = lower(email) AND char_length(email) <= 254),
  password_hash TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sessions (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash   TEXT NOT NULL UNIQUE,
  user_agent   TEXT NULL,
  ip_address   INET NULL,
  expires_at   TIMESTAMPTZ NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_used_at TIMESTAMPTZ NULL,
  revoked_at   TIMESTAMPTZ NULL
);
CREATE INDEX idx_sessions_user ON sessions (user_id);

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

-- ---------------------------------------------------------------- profiles
CREATE TABLE profiles (
  user_id      UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name   TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date   DATE NOT NULL,
  gender       TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'nonbinary')),
  bio          TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city         TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coarsened server-side to a ~1 km grid; exact positions are never stored.
  latitude     DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude    DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_discovery ON profiles (gender, birth_date) WHERE discoverable;
CREATE INDEX idx_profiles_geo ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in   TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'nonbinary']
  ),
  min_age         SMALLINT NOT NULL DEFAULT 18 CHECK (min_age BETWEEN 18 AND 99),
  max_age         SMALLINT NOT NULL DEFAULT 99 CHECK (max_age BETWEEN 18 AND 99),
  max_distance_km SMALLINT NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (min_age <= max_age)
);

CREATE TABLE interests (
  id       SMALLSERIAL PRIMARY KEY,
  slug     TEXT NOT NULL UNIQUE,
  label    TEXT NOT NULL,
  category TEXT NOT NULL
);

CREATE TABLE user_interests (
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id);

CREATE TABLE photos (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position    SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  storage_key TEXT NOT NULL UNIQUE,
  width       INTEGER NOT NULL,
  height      INTEGER NOT NULL,
  size_bytes  INTEGER NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_photos_owner_created ON photos (user_id, created_at);

-- ---------------------------------------------------------------- matching
CREATE TABLE swipes (
  swiper_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action     TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CHECK (swiper_id <> target_id)
);
CREATE INDEX idx_swipes_target_likes ON swipes (target_id, swiper_id) WHERE action = 'like';

-- A match is the conversation: one row per unordered pair (user_a < user_b).
CREATE TABLE matches (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (user_a < user_b),
  UNIQUE (user_a, user_b)
);
CREATE INDEX idx_matches_user_b ON matches (user_b, user_a);

CREATE TABLE match_participants (
  match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  cleared_at TIMESTAMPTZ NULL,
  PRIMARY KEY (match_id, user_id)
);
CREATE INDEX idx_match_participants_user ON match_participants (user_id, match_id);

CREATE TABLE messages (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body       TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  read_at    TIMESTAMPTZ NULL
);
CREATE INDEX idx_messages_match_created ON messages (match_id, created_at DESC, id DESC);
CREATE INDEX idx_messages_unread ON messages (match_id, sender_id) WHERE read_at IS NULL;

CREATE TABLE notifications (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type       TEXT NOT NULL CHECK (type IN ('match', 'message')),
  actor_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  read_at    TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_unread ON notifications (user_id) WHERE read_at IS NULL;

-- ---------------------------------------------------------------- safety
CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id, blocker_id);

-- reported_id is nulled (not cascaded) when the reported account is deleted so moderation
-- evidence survives account deletion used to dodge a report.
CREATE TABLE reports (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  match_id    UUID NULL REFERENCES matches(id) ON DELETE SET NULL,
  reason      TEXT NOT NULL CHECK (reason IN ('spam', 'fake_profile', 'harassment', 'inappropriate_content', 'underage', 'other')),
  details     TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reports_status_created ON reports (status, created_at);
CREATE INDEX idx_reports_reported ON reports (reported_id);

-- ---------------------------------------------------------------- helpers
CREATE FUNCTION haversine_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION LANGUAGE SQL IMMUTABLE PARALLEL SAFE AS $$
  SELECT 6371.0 * acos(least(1.0, greatest(-1.0,
    sin(radians(lat1)) * sin(radians(lat2)) +
    cos(radians(lat1)) * cos(radians(lat2)) * cos(radians(lon2 - lon1)))))
$$;

-- Reference data (not demo data): the selectable interest catalogue.
INSERT INTO interests (slug, label, category) VALUES
  ('hiking', 'Hiking', 'outdoors'), ('camping', 'Camping', 'outdoors'), ('cycling', 'Cycling', 'outdoors'),
  ('running', 'Running', 'sport'), ('yoga', 'Yoga', 'sport'), ('climbing', 'Climbing', 'sport'),
  ('football', 'Football', 'sport'), ('swimming', 'Swimming', 'sport'),
  ('cooking', 'Cooking', 'food'), ('coffee', 'Coffee', 'food'), ('wine', 'Wine', 'food'), ('baking', 'Baking', 'food'),
  ('travel', 'Travel', 'lifestyle'), ('photography', 'Photography', 'creative'), ('painting', 'Painting', 'creative'),
  ('writing', 'Writing', 'creative'), ('music', 'Music', 'culture'), ('concerts', 'Concerts', 'culture'),
  ('cinema', 'Cinema', 'culture'), ('theatre', 'Theatre', 'culture'), ('museums', 'Museums', 'culture'),
  ('reading', 'Reading', 'culture'), ('gaming', 'Gaming', 'fun'), ('board-games', 'Board games', 'fun'),
  ('dancing', 'Dancing', 'fun'), ('pets', 'Pets', 'lifestyle'), ('gardening', 'Gardening', 'lifestyle'),
  ('volunteering', 'Volunteering', 'lifestyle'), ('tech', 'Tech', 'lifestyle'), ('languages', 'Languages', 'culture');
