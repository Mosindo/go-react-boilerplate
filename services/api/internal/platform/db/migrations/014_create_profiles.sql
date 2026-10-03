CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coordinates are stored already rounded to ~1 km by the service layer.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_profiles_discoverable_gender
  ON profiles (gender, birth_date) WHERE is_visible;

CREATE INDEX IF NOT EXISTS idx_profiles_location
  ON profiles (latitude, longitude) WHERE is_visible AND latitude IS NOT NULL;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['man', 'woman', 'nonbinary']
    CHECK (cardinality(interested_in) BETWEEN 1 AND 3
           AND interested_in <@ ARRAY['man', 'woman', 'nonbinary']),
  min_age SMALLINT NOT NULL DEFAULT 18 CHECK (min_age BETWEEN 18 AND 99),
  max_age SMALLINT NOT NULL DEFAULT 99 CHECK (max_age BETWEEN 18 AND 99),
  -- NULL means "anywhere".
  max_distance_km INTEGER NULL CHECK (max_distance_km BETWEEN 1 AND 500),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (min_age <= max_age)
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Travel'), ('cooking', 'Cooking'), ('hiking', 'Hiking'),
  ('music', 'Music'), ('cinema', 'Cinema'), ('reading', 'Reading'),
  ('sports', 'Sports'), ('yoga', 'Yoga'), ('photography', 'Photography'),
  ('gaming', 'Gaming'), ('art', 'Art'), ('dancing', 'Dancing'),
  ('coffee', 'Coffee'), ('wine', 'Wine'), ('pets', 'Pets'),
  ('nature', 'Nature'), ('technology', 'Technology'), ('fashion', 'Fashion'),
  ('theatre', 'Theatre'), ('festivals', 'Festivals'), ('cycling', 'Cycling'),
  ('running', 'Running'), ('volunteering', 'Volunteering'), ('board-games', 'Board games')
ON CONFLICT (slug) DO NOTHING;

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
  storage_key TEXT NOT NULL UNIQUE,
  content_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_user_position_key UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
