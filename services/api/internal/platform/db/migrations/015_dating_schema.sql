-- Profiles ------------------------------------------------------------------
CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coarse coordinates only: the API rounds them to 0.01 degree (~1 km).
  latitude DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_profiles_discovery ON profiles (latitude, longitude) WHERE discoverable;
CREATE INDEX idx_profiles_birth_date ON profiles (birth_date);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'nonbinary']::TEXT[]
  ),
  min_age SMALLINT NOT NULL CHECK (min_age BETWEEN 18 AND 99),
  max_age SMALLINT NOT NULL CHECK (max_age BETWEEN 18 AND 99),
  max_distance_km INTEGER NOT NULL CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (min_age <= max_age)
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

-- Photos --------------------------------------------------------------------
CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  mime_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  width INTEGER NOT NULL CHECK (width > 0),
  height INTEGER NOT NULL CHECK (height > 0),
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);

CREATE INDEX idx_photos_user_created_at ON photos (user_id, created_at);

-- Matching ------------------------------------------------------------------
CREATE TABLE swipes (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, target_id),
  CHECK (user_id <> target_id)
);

CREATE INDEX idx_swipes_target_action ON swipes (target_id, action, created_at DESC);

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unmatched_at TIMESTAMPTZ NULL,
  -- Canonical ordering guarantees one row per pair.
  CHECK (user_a < user_b),
  UNIQUE (user_a, user_b)
);

CREATE INDEX idx_matches_user_a ON matches (user_a, created_at DESC);
CREATE INDEX idx_matches_user_b ON matches (user_b, created_at DESC);

-- Chat ----------------------------------------------------------------------
CREATE TABLE conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_message_at TIMESTAMPTZ NULL
);

CREATE TABLE conversation_participants (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  -- "Delete conversation" is local: history before cleared_at is hidden for this user only.
  cleared_at TIMESTAMPTZ NULL,
  hidden BOOLEAN NOT NULL DEFAULT FALSE,
  PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, conversation_id);

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX idx_messages_conversation_created_at ON messages (conversation_id, created_at DESC, id DESC);

-- Safety --------------------------------------------------------------------
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
  -- SET NULL keeps the report for moderators even if an account is later deleted.
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL CHECK (reason IN ('spam', 'fake', 'harassment', 'inappropriate_photos', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE INDEX idx_reports_reporter_created_at ON reports (reporter_id, created_at DESC);
CREATE INDEX idx_reports_status ON reports (status, created_at DESC);

-- Account recovery ----------------------------------------------------------
CREATE TABLE password_reset_codes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_codes_user ON password_reset_codes (user_id, created_at DESC);

-- Notifications (table created by 005): in-app notifications are coalesced by dedupe_key.
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS dedupe_key TEXT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_unread_dedupe
  ON notifications (user_id, dedupe_key)
  WHERE is_read = FALSE AND dedupe_key IS NOT NULL;

-- Curated interest catalogue --------------------------------------------------
INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyages'),
  ('hiking', 'Randonnée'),
  ('cooking', 'Cuisine'),
  ('coffee', 'Café'),
  ('wine', 'Vin'),
  ('restaurants', 'Restaurants'),
  ('movies', 'Cinéma'),
  ('series', 'Séries'),
  ('reading', 'Lecture'),
  ('music', 'Musique'),
  ('concerts', 'Concerts'),
  ('dancing', 'Danse'),
  ('photography', 'Photographie'),
  ('art', 'Art'),
  ('museums', 'Musées'),
  ('theatre', 'Théâtre'),
  ('gaming', 'Jeux vidéo'),
  ('board-games', 'Jeux de société'),
  ('running', 'Course à pied'),
  ('yoga', 'Yoga'),
  ('gym', 'Musculation'),
  ('cycling', 'Vélo'),
  ('swimming', 'Natation'),
  ('football', 'Football'),
  ('tennis', 'Tennis'),
  ('climbing', 'Escalade'),
  ('skiing', 'Ski'),
  ('surfing', 'Surf'),
  ('nature', 'Nature'),
  ('gardening', 'Jardinage'),
  ('animals', 'Animaux'),
  ('volunteering', 'Bénévolat'),
  ('technology', 'Technologie'),
  ('science', 'Sciences'),
  ('fashion', 'Mode'),
  ('writing', 'Écriture'),
  ('diy', 'Bricolage'),
  ('podcasts', 'Podcasts'),
  ('meditation', 'Méditation'),
  ('languages', 'Langues')
ON CONFLICT (slug) DO NOTHING;
