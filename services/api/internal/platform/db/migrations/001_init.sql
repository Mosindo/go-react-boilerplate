-- Dating app schema. Every user-owned table cascades on user deletion so that
-- DELETE FROM users removes all associated personal data.

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  is_demo BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT users_email_normalized CHECK (email = lower(btrim(email)) AND char_length(email) <= 254)
);

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
CREATE INDEX idx_sessions_user_created_at ON sessions (user_id, created_at DESC, id DESC);

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

CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coordinates are rounded to 2 decimals (~1 km) before storage and never returned by the API.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_discovery ON profiles (gender, birth_date) WHERE discoverable;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  min_age SMALLINT NOT NULL CHECK (min_age >= 18),
  max_age SMALLINT NOT NULL CHECK (max_age <= 99),
  max_distance_km INTEGER NULL CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (max_age >= min_age),
  CONSTRAINT preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'nonbinary']::TEXT[]
  )
);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  content BYTEA NOT NULL,
  thumb BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
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

CREATE TABLE swipes (
  id BIGSERIAL PRIMARY KEY,
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT swipes_unique_pair UNIQUE (from_user_id, to_user_id),
  CONSTRAINT swipes_no_self CHECK (from_user_id <> to_user_id)
);
CREATE INDEX idx_swipes_to_user ON swipes (to_user_id, action);

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  -- Canonical ordering guarantees a single row per pair.
  CONSTRAINT matches_ordered_pair CHECK (user_a < user_b),
  CONSTRAINT matches_unique_pair UNIQUE (user_a, user_b)
);
CREATE INDEX idx_matches_user_b ON matches (user_b, created_at DESC);
CREATE INDEX idx_matches_user_a_created ON matches (user_a, created_at DESC);

CREATE TABLE conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_participants (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  hidden_at TIMESTAMPTZ NULL,
  PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, conversation_id);

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  read_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_messages_conversation_created ON messages (conversation_id, created_at DESC, id DESC);
CREATE INDEX idx_messages_unread ON messages (conversation_id, sender_id) WHERE read_at IS NULL;

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT blocks_no_self CHECK (blocker_id <> blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks (blocked_id);

CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL CHECK (reason IN ('spam', 'fake_profile', 'harassment', 'inappropriate_content', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT reports_no_self CHECK (reporter_id <> reported_id)
);
CREATE INDEX idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE INDEX idx_reports_status ON reports (status, created_at DESC);
CREATE INDEX idx_reports_reporter ON reports (reporter_id, created_at DESC);

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL CHECK (type IN ('match', 'message')),
  actor_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id UUID NULL REFERENCES conversations(id) ON DELETE CASCADE,
  read_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_unread ON notifications (user_id) WHERE read_at IS NULL;
-- At most one unread message notification per conversation and recipient.
CREATE UNIQUE INDEX idx_notifications_message_coalesce
  ON notifications (user_id, conversation_id)
  WHERE type = 'message' AND read_at IS NULL;

-- Reference data (not demo data): interest catalog.
INSERT INTO interests (slug, label) VALUES
  ('travel', 'Voyages'), ('cooking', 'Cuisine'), ('music', 'Musique'), ('cinema', 'Cinéma'),
  ('reading', 'Lecture'), ('hiking', 'Randonnée'), ('sports', 'Sport'), ('yoga', 'Yoga'),
  ('photography', 'Photographie'), ('art', 'Art'), ('gaming', 'Jeux vidéo'), ('dancing', 'Danse'),
  ('theatre', 'Théâtre'), ('wine', 'Vin'), ('coffee', 'Café'), ('animals', 'Animaux'),
  ('gardening', 'Jardinage'), ('cycling', 'Vélo'), ('running', 'Course à pied'), ('swimming', 'Natation'),
  ('tech', 'Technologie'), ('fashion', 'Mode'), ('volunteering', 'Bénévolat'), ('board-games', 'Jeux de société'),
  ('concerts', 'Concerts'), ('museums', 'Musées'), ('camping', 'Camping'), ('languages', 'Langues'),
  ('podcasts', 'Podcasts'), ('diy', 'Bricolage');
