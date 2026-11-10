-- Lueur: initial dating schema.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------------------
-- Accounts & sessions
-- ---------------------------------------------------------------------------
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  is_demo BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_users_email_normalized CHECK (email = lower(btrim(email)) AND char_length(email) BETWEEN 3 AND 254)
);

CREATE UNIQUE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_demo ON users (is_demo) WHERE is_demo;

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
CREATE INDEX idx_sessions_user_created_at ON sessions (user_id, created_at DESC);

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

-- ---------------------------------------------------------------------------
-- Profiles, preferences, interests, photos
-- ---------------------------------------------------------------------------
CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  height_cm SMALLINT NULL,
  relationship_goal TEXT NULL,
  -- Coarse location only (rounded server side to ~1 km). Never exposed raw.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  is_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  moderation_status TEXT NOT NULL DEFAULT 'active',
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_first_name CHECK (char_length(btrim(first_name)) BETWEEN 1 AND 30),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_job_title CHECK (char_length(job_title) <= 60),
  CONSTRAINT chk_profiles_city CHECK (char_length(city) <= 60),
  CONSTRAINT chk_profiles_height CHECK (height_cm IS NULL OR height_cm BETWEEN 120 AND 230),
  CONSTRAINT chk_profiles_goal CHECK (relationship_goal IS NULL OR relationship_goal IN ('long_term', 'short_term', 'friendship', 'undecided')),
  CONSTRAINT chk_profiles_location CHECK (
    (latitude IS NULL AND longitude IS NULL)
    OR (latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180)
  ),
  CONSTRAINT chk_profiles_moderation CHECK (moderation_status IN ('active', 'under_review', 'suspended'))
);

CREATE INDEX idx_profiles_discovery_geo
  ON profiles (latitude, longitude)
  WHERE is_discoverable AND moderation_status = 'active' AND latitude IS NOT NULL;
CREATE INDEX idx_profiles_gender_birthdate ON profiles (gender, birthdate);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  ),
  CONSTRAINT chk_preferences_ages CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
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
  position SMALLINT NOT NULL,
  storage_key TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);

CREATE UNIQUE INDEX idx_photos_storage_key ON photos (storage_key);

-- ---------------------------------------------------------------------------
-- Matching
-- ---------------------------------------------------------------------------
CREATE TABLE swipes (
  swiper_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (swiper_id, target_id),
  CONSTRAINT chk_swipes_action CHECK (action IN ('like', 'pass')),
  CONSTRAINT chk_swipes_not_self CHECK (swiper_id <> target_id)
);

CREATE INDEX idx_swipes_target_likes ON swipes (target_id, swiper_id) WHERE action = 'like';

-- A match is the 1:1 conversation between two people (user_a_id < user_b_id).
CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_matches_ordered CHECK (user_a_id < user_b_id),
  CONSTRAINT uq_matches_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX idx_matches_user_a_activity ON matches (user_a_id, last_activity_at DESC, id DESC);
CREATE INDEX idx_matches_user_b_activity ON matches (user_b_id, last_activity_at DESC, id DESC);

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_messages_body CHECK (char_length(btrim(body)) BETWEEN 1 AND 2000)
);

CREATE INDEX idx_messages_match_created_at ON messages (match_id, created_at DESC, id DESC);
CREATE INDEX idx_messages_unread ON messages (match_id, sender_id) WHERE read_at IS NULL;

-- ---------------------------------------------------------------------------
-- Safety & moderation
-- ---------------------------------------------------------------------------
CREATE TABLE blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT chk_blocks_not_self CHECK (blocker_id <> blocked_id)
);

CREATE INDEX idx_blocks_blocked ON blocks (blocked_id, blocker_id);

CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reported_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  details TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_reports_reason CHECK (reason IN ('fake_profile', 'inappropriate_content', 'harassment', 'scam', 'underage', 'other')),
  CONSTRAINT chk_reports_details CHECK (char_length(details) <= 1000),
  CONSTRAINT chk_reports_status CHECK (status IN ('open', 'dismissed', 'actioned')),
  CONSTRAINT uq_reports_pair UNIQUE (reporter_id, reported_id)
);

CREATE INDEX idx_reports_reported_status ON reports (reported_id, status);
CREATE INDEX idx_reports_status_created_at ON reports (status, created_at DESC);

-- ---------------------------------------------------------------------------
-- Notifications
-- ---------------------------------------------------------------------------
CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  actor_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  match_id UUID NULL REFERENCES matches(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL,
  CONSTRAINT chk_notifications_type CHECK (type IN ('match', 'message'))
);

CREATE INDEX idx_notifications_user_created_at ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_user_unread ON notifications (user_id) WHERE read_at IS NULL;
-- At most one unread "message" notification per conversation (collapsed).
CREATE UNIQUE INDEX idx_notifications_unread_message_per_match
  ON notifications (user_id, match_id)
  WHERE type = 'message' AND read_at IS NULL;

-- ---------------------------------------------------------------------------
-- Reference data: interests catalog (not demo data).
-- ---------------------------------------------------------------------------
INSERT INTO interests (id, slug, label) VALUES
  (1, 'hiking', 'Randonnée'),
  (2, 'running', 'Course à pied'),
  (3, 'yoga', 'Yoga'),
  (4, 'climbing', 'Escalade'),
  (5, 'cycling', 'Vélo'),
  (6, 'swimming', 'Natation'),
  (7, 'football', 'Football'),
  (8, 'tennis', 'Tennis'),
  (9, 'dance', 'Danse'),
  (10, 'cooking', 'Cuisine'),
  (11, 'baking', 'Pâtisserie'),
  (12, 'wine', 'Vin'),
  (13, 'coffee', 'Café'),
  (14, 'brunch', 'Brunch'),
  (15, 'vegetarian', 'Végétarien'),
  (16, 'travel', 'Voyages'),
  (17, 'roadtrips', 'Road trips'),
  (18, 'camping', 'Camping'),
  (19, 'beach', 'Plage'),
  (20, 'mountains', 'Montagne'),
  (21, 'music', 'Musique'),
  (22, 'concerts', 'Concerts'),
  (23, 'festivals', 'Festivals'),
  (24, 'cinema', 'Cinéma'),
  (25, 'series', 'Séries'),
  (26, 'theatre', 'Théâtre'),
  (27, 'reading', 'Lecture'),
  (28, 'writing', 'Écriture'),
  (29, 'photography', 'Photographie'),
  (30, 'painting', 'Peinture'),
  (31, 'museums', 'Musées'),
  (32, 'design', 'Design'),
  (33, 'fashion', 'Mode'),
  (34, 'gaming', 'Jeux vidéo'),
  (35, 'boardgames', 'Jeux de société'),
  (36, 'tech', 'Tech'),
  (37, 'science', 'Sciences'),
  (38, 'languages', 'Langues'),
  (39, 'volunteering', 'Bénévolat'),
  (40, 'gardening', 'Jardinage'),
  (41, 'animals', 'Animaux'),
  (42, 'dogs', 'Chiens'),
  (43, 'cats', 'Chats'),
  (44, 'meditation', 'Méditation'),
  (45, 'fitness', 'Fitness'),
  (46, 'podcasts', 'Podcasts'),
  (47, 'karaoke', 'Karaoké'),
  (48, 'nightlife', 'Sorties');
