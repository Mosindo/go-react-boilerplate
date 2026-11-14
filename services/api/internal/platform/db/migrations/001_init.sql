-- Lueur dating schema (v1).
-- Every table that belongs to a user cascades on account deletion so that
-- deleting a user removes all of their personal data in one statement.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Great-circle distance in kilometres (haversine), used by discovery.
CREATE OR REPLACE FUNCTION distance_km(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION)
RETURNS DOUBLE PRECISION
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE AS $$
  SELECT 12742 * asin(sqrt(
    power(sin(radians(lat2 - lat1) / 2), 2)
    + cos(radians(lat1)) * cos(radians(lat2)) * power(sin(radians(lon2 - lon1) / 2), 2)
  ))
$$;

-- Age in whole years at the current date.
CREATE OR REPLACE FUNCTION age_years(birthdate DATE)
RETURNS INTEGER
LANGUAGE sql STABLE STRICT PARALLEL SAFE AS $$
  SELECT date_part('year', age(CURRENT_DATE, birthdate))::INTEGER
$$;

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  is_demo BOOLEAN NOT NULL DEFAULT FALSE,
  suspended_at TIMESTAMPTZ NULL,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT users_email_lowercase CHECK (email = lower(btrim(email)))
);

CREATE UNIQUE INDEX idx_users_email ON users (email);

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
CREATE INDEX idx_sessions_user_id ON sessions (user_id);

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
  first_name TEXT NOT NULL,
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  height_cm SMALLINT NULL,
  looking_for TEXT NOT NULL DEFAULT 'not_sure',
  city TEXT NOT NULL DEFAULT '',
  -- Coordinates are coarsened to 2 decimals (~1 km) before storage and are
  -- never returned by the API; only an approximate distance is exposed.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_gender_valid CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT profiles_looking_for_valid CHECK (looking_for IN ('relationship', 'casual', 'friendship', 'not_sure')),
  CONSTRAINT profiles_first_name_length CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT profiles_bio_length CHECK (char_length(bio) <= 500),
  CONSTRAINT profiles_job_title_length CHECK (char_length(job_title) <= 80),
  CONSTRAINT profiles_city_length CHECK (char_length(city) <= 80),
  CONSTRAINT profiles_height_range CHECK (height_cm IS NULL OR height_cm BETWEEN 100 AND 250),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL)),
  CONSTRAINT profiles_latitude_range CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT profiles_longitude_range CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180)
);

CREATE INDEX idx_profiles_discovery ON profiles (gender, birthdate) WHERE discoverable;
CREATE INDEX idx_profiles_location ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_interested_in_valid CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  ),
  CONSTRAINT preferences_age_range CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT preferences_distance_range CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  category TEXT NOT NULL
);

CREATE TABLE user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX idx_user_interests_interest ON user_interests (interest_id, user_id);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  mime_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  position SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_position_range CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT photos_mime_type_valid CHECK (mime_type = 'image/jpeg'),
  CONSTRAINT photos_user_position_unique UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- One decision (like or pass) per ordered pair; the primary key prevents
-- duplicate or repeated likes and keeps processed profiles out of discovery.
CREATE TABLE swipes (
  actor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (actor_id, target_id),
  CONSTRAINT swipes_action_valid CHECK (action IN ('like', 'pass')),
  CONSTRAINT swipes_not_self CHECK (actor_id <> target_id)
);

CREATE INDEX idx_swipes_target_action ON swipes (target_id, action, actor_id);

-- A match is stored once per unordered pair (user_a_id < user_b_id).
-- The match doubles as the conversation between both users.
CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_message_at TIMESTAMPTZ NULL,
  CONSTRAINT matches_ordered_pair CHECK (user_a_id < user_b_id),
  CONSTRAINT matches_pair_unique UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX idx_matches_user_b ON matches (user_b_id);

CREATE TABLE conversation_participants (
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_at TIMESTAMPTZ NULL,
  cleared_at TIMESTAMPTZ NULL,
  PRIMARY KEY (match_id, user_id)
);

CREATE INDEX idx_conversation_participants_user ON conversation_participants (user_id, match_id);

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT messages_body_length CHECK (char_length(body) BETWEEN 1 AND 2000)
);

CREATE INDEX idx_messages_match_created_at ON messages (match_id, created_at DESC, id DESC);

CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT blocks_not_self CHECK (blocker_id <> blocked_id)
);

CREATE INDEX idx_blocks_blocked ON blocks (blocked_id, blocker_id);

-- Reports outlive account deletion (SET NULL) so moderators keep a trace.
CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ NULL,
  CONSTRAINT reports_reason_valid CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'spam', 'underage', 'other')),
  CONSTRAINT reports_status_valid CHECK (status IN ('open', 'reviewed', 'dismissed')),
  CONSTRAINT reports_details_length CHECK (char_length(details) <= 1000)
);

CREATE UNIQUE INDEX idx_reports_open_pair ON reports (reporter_id, reported_id) WHERE status = 'open';
CREATE INDEX idx_reports_reported_status ON reports (reported_id, status);

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  is_read BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL,
  CONSTRAINT notifications_type_valid CHECK (type IN ('match', 'message', 'system'))
);

CREATE INDEX idx_notifications_user_created_at ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_user_unread ON notifications (user_id) WHERE NOT is_read;

-- Reference catalogue (not demo data): interests users can pick from.
INSERT INTO interests (slug, label, category) VALUES
  ('hiking', 'Randonnée', 'Sport'),
  ('running', 'Course à pied', 'Sport'),
  ('yoga', 'Yoga', 'Sport'),
  ('climbing', 'Escalade', 'Sport'),
  ('cycling', 'Vélo', 'Sport'),
  ('swimming', 'Natation', 'Sport'),
  ('football', 'Football', 'Sport'),
  ('dance', 'Danse', 'Sport'),
  ('cooking', 'Cuisine', 'Art de vivre'),
  ('wine', 'Vin', 'Art de vivre'),
  ('coffee', 'Café', 'Art de vivre'),
  ('brunch', 'Brunch', 'Art de vivre'),
  ('gardening', 'Jardinage', 'Art de vivre'),
  ('travel', 'Voyages', 'Art de vivre'),
  ('animals', 'Animaux', 'Art de vivre'),
  ('volunteering', 'Bénévolat', 'Art de vivre'),
  ('music', 'Musique', 'Culture'),
  ('concerts', 'Concerts', 'Culture'),
  ('cinema', 'Cinéma', 'Culture'),
  ('reading', 'Lecture', 'Culture'),
  ('theatre', 'Théâtre', 'Culture'),
  ('museums', 'Musées', 'Culture'),
  ('photography', 'Photographie', 'Culture'),
  ('art', 'Arts plastiques', 'Culture'),
  ('writing', 'Écriture', 'Culture'),
  ('video_games', 'Jeux vidéo', 'Loisirs'),
  ('board_games', 'Jeux de société', 'Loisirs'),
  ('series', 'Séries', 'Loisirs'),
  ('tech', 'Tech', 'Loisirs'),
  ('science', 'Sciences', 'Loisirs'),
  ('meditation', 'Méditation', 'Bien-être'),
  ('nature', 'Nature', 'Bien-être'),
  ('sea', 'Mer', 'Bien-être'),
  ('mountain', 'Montagne', 'Bien-être'),
  ('languages', 'Langues', 'Loisirs'),
  ('fashion', 'Mode', 'Culture');
