-- Dating product schema.
-- Replaces the generic boilerplate domain tables (posts, comments, files,
-- billing, multi-tenancy, legacy chat) with the dating domain.

DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS tenant_users CASCADE;
DROP TABLE IF EXISTS votes CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS files CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS conversation_participants CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_organization_id_fkey;
ALTER TABLE users DROP COLUMN IF EXISTS organization_id;
ALTER TABLE users DROP COLUMN IF EXISTS display_name;
ALTER TABLE users DROP COLUMN IF EXISTS bio;
DROP TABLE IF EXISTS organizations CASCADE;

ALTER TABLE users ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_lowercase;
ALTER TABLE users ADD CONSTRAINT users_email_lowercase CHECK (email = lower(email));

CREATE TABLE IF NOT EXISTS password_resets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_password_resets_user ON password_resets (user_id, created_at DESC);

-- Profiles ------------------------------------------------------------------

CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NULL CHECK (city IS NULL OR char_length(city) <= 80),
  -- Coarse location: coordinates are snapped to ~1 km before being stored.
  lat DOUBLE PRECISION NULL CHECK (lat BETWEEN -90 AND 90),
  lng DOUBLE PRECISION NULL CHECK (lng BETWEEN -180 AND 180),
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((lat IS NULL) = (lng IS NULL))
);
CREATE INDEX idx_profiles_discovery ON profiles (is_visible, gender);
CREATE INDEX idx_profiles_location ON profiles (lat, lng) WHERE lat IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'non_binary']
  ),
  min_age SMALLINT NOT NULL DEFAULT 18 CHECK (min_age BETWEEN 18 AND 99),
  max_age SMALLINT NOT NULL DEFAULT 99 CHECK (max_age BETWEEN 18 AND 99),
  max_distance_km INTEGER NULL CHECK (max_distance_km IS NULL OR max_distance_km BETWEEN 1 AND 500),
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (max_age >= min_age)
);

CREATE TABLE interests (
  id SMALLINT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO interests (id, slug, label) VALUES
  (1, 'travel', 'Voyages'),
  (2, 'cooking', 'Cuisine'),
  (3, 'music', 'Musique'),
  (4, 'cinema', 'Cinéma'),
  (5, 'reading', 'Lecture'),
  (6, 'hiking', 'Randonnée'),
  (7, 'sport', 'Sport'),
  (8, 'yoga', 'Yoga'),
  (9, 'photography', 'Photographie'),
  (10, 'art', 'Art'),
  (11, 'gaming', 'Jeux vidéo'),
  (12, 'board_games', 'Jeux de société'),
  (13, 'dancing', 'Danse'),
  (14, 'theatre', 'Théâtre'),
  (15, 'coffee', 'Café'),
  (16, 'wine', 'Vin'),
  (17, 'animals', 'Animaux'),
  (18, 'gardening', 'Jardinage'),
  (19, 'running', 'Course à pied'),
  (20, 'cycling', 'Vélo'),
  (21, 'technology', 'Technologie'),
  (22, 'science', 'Sciences'),
  (23, 'history', 'Histoire'),
  (24, 'fashion', 'Mode'),
  (25, 'festivals', 'Festivals'),
  (26, 'volunteering', 'Bénévolat'),
  (27, 'swimming', 'Natation'),
  (28, 'climbing', 'Escalade'),
  (29, 'writing', 'Écriture'),
  (30, 'podcasts', 'Podcasts')
ON CONFLICT (id) DO NOTHING;

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
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 8),
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);
CREATE INDEX idx_photos_user_position ON photos (user_id, position);

-- Matching ------------------------------------------------------------------

CREATE TABLE swipes (
  from_user UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (from_user, to_user),
  CHECK (from_user <> to_user)
);
CREATE INDEX idx_swipes_incoming_likes ON swipes (to_user, created_at DESC) WHERE action = 'like';

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_a, user_b),
  CHECK (user_a < user_b)
);
CREATE INDEX idx_matches_user_b ON matches (user_b);

-- Chat ----------------------------------------------------------------------

CREATE TABLE conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_message_at TIMESTAMPTZ NULL
);

CREATE TABLE conversation_members (
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  hidden_at TIMESTAMPTZ NULL,
  PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_conversation_members_user ON conversation_members (user_id);

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX idx_messages_conversation_created ON messages (conversation_id, created_at DESC, id DESC);

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
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL CHECK (reason IN ('spam', 'fake_profile', 'harassment', 'inappropriate_content', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  -- Snapshot of the last messages exchanged, kept so moderators have evidence
  -- even if the reporter later blocks (and thereby removes) the conversation.
  context JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (reporter_id <> reported_id)
);
CREATE INDEX idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE INDEX idx_reports_open ON reports (created_at DESC) WHERE status = 'open';

-- Notifications ---------------------------------------------------------------

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL CHECK (type IN ('new_match', 'new_message')),
  actor_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  conversation_id UUID NULL REFERENCES conversations(id) ON DELETE CASCADE,
  read_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_unread ON notifications (user_id) WHERE read_at IS NULL;
