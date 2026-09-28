-- Dating app core schema. Legacy tables that are incompatible (conversations,
-- messages, conversation_participants) are left untouched; the dating chat uses
-- match_conversations / match_messages instead.

-- Users no longer belong to an organization.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_organization_id_fkey;
DROP INDEX IF EXISTS idx_users_organization_id_created_at;
ALTER TABLE users DROP COLUMN IF EXISTS organization_id;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_email_lowercase') THEN
    ALTER TABLE users ADD CONSTRAINT users_email_lowercase CHECK (email = lower(email)) NOT VALID;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 50),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  location_label TEXT NOT NULL DEFAULT '' CHECK (char_length(location_label) <= 80),
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  is_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_profiles_lat_lon
  ON profiles (latitude, longitude)
  WHERE is_discoverable AND latitude IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_profiles_discover
  ON profiles (gender, birth_date)
  WHERE is_discoverable;
CREATE INDEX IF NOT EXISTS idx_profiles_last_active
  ON profiles (last_active_at DESC);

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['man', 'woman', 'non_binary'],
  min_age SMALLINT NOT NULL DEFAULT 18 CHECK (min_age BETWEEN 18 AND 99),
  max_age SMALLINT NOT NULL DEFAULT 99 CHECK (max_age BETWEEN 18 AND 99),
  max_distance_km INTEGER NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (max_age >= min_age),
  CONSTRAINT preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'non_binary']
  )
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9_]+$'),
  label TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);
CREATE INDEX IF NOT EXISTS idx_user_interests_interest ON user_interests (interest_id, user_id);

CREATE TABLE IF NOT EXISTS photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  content BYTEA NOT NULL,
  byte_size INTEGER NOT NULL CHECK (byte_size > 0),
  width INTEGER NOT NULL CHECK (width > 0),
  height INTEGER NOT NULL CHECK (height > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX IF NOT EXISTS idx_photos_user_created_at ON photos (user_id, created_at);

CREATE TABLE IF NOT EXISTS swipes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL CHECK (action IN ('like', 'pass')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT swipes_no_self CHECK (from_user_id <> to_user_id),
  CONSTRAINT swipes_pair_key UNIQUE (from_user_id, to_user_id)
);
CREATE INDEX IF NOT EXISTS idx_swipes_to_user ON swipes (to_user_id, from_user_id);
CREATE INDEX IF NOT EXISTS idx_swipes_from_created_at ON swipes (from_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT matches_ordered CHECK (user_a < user_b),
  CONSTRAINT matches_pair_key UNIQUE (user_a, user_b)
);
CREATE INDEX IF NOT EXISTS idx_matches_user_b ON matches (user_b, user_a);

CREATE TABLE IF NOT EXISTS match_conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_message_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS match_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES match_conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_match_messages_conversation_created
  ON match_messages (conversation_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_match_messages_unread
  ON match_messages (conversation_id, sender_id)
  WHERE read_at IS NULL;

CREATE TABLE IF NOT EXISTS blocks (
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (blocker_id, blocked_id),
  CONSTRAINT blocks_no_self CHECK (blocker_id <> blocked_id)
);
CREATE INDEX IF NOT EXISTS idx_blocks_blocked ON blocks (blocked_id, blocker_id);

CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL CHECK (reason IN ('spam', 'fake_profile', 'harassment', 'inappropriate_content', 'underage', 'other')),
  details TEXT NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT reports_no_self CHECK (reporter_id <> reported_id)
);
CREATE INDEX IF NOT EXISTS idx_reports_reported ON reports (reported_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_reporter_created ON reports (reporter_id, created_at DESC);

CREATE TABLE IF NOT EXISTS password_resets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  attempts SMALLINT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_password_resets_user_created ON password_resets (user_id, created_at DESC);

-- notifications: reuse the legacy table. read_at is the single source of truth.
DROP INDEX IF EXISTS idx_notifications_user_is_read_created_at;
DROP INDEX IF EXISTS idx_notifications_user_unread_created_at;
ALTER TABLE notifications DROP COLUMN IF EXISTS is_read;
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
  ON notifications (user_id, created_at DESC, id DESC)
  WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_data_match
  ON notifications ((data ->> 'matchId'))
  WHERE data ? 'matchId';
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'notifications_type_dating') THEN
    ALTER TABLE notifications ADD CONSTRAINT notifications_type_dating CHECK (type IN ('match', 'message')) NOT VALID;
  END IF;
END $$;
