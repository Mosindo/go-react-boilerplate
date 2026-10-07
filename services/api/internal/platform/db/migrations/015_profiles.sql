CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 40),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Location is stored rounded to ~1 km (2 decimals): never the exact position.
  latitude DOUBLE PRECISION CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION CHECK (longitude BETWEEN -180 AND 180),
  location_updated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX idx_profiles_birth_date ON profiles (birth_date);
CREATE INDEX idx_profiles_location ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['man', 'woman', 'nonbinary'],
  min_age SMALLINT NOT NULL DEFAULT 18 CHECK (min_age BETWEEN 18 AND 99),
  max_age SMALLINT NOT NULL DEFAULT 99 CHECK (max_age BETWEEN 18 AND 99),
  max_distance_km SMALLINT NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_age_range CHECK (min_age <= max_age),
  CONSTRAINT chk_preferences_interested CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'nonbinary']
  )
);

CREATE TABLE interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

-- Reference catalog (not demo data).
INSERT INTO interests (slug, label) VALUES
  ('hiking', 'Hiking'), ('travel', 'Travel'), ('cooking', 'Cooking'), ('coffee', 'Coffee'),
  ('wine', 'Wine'), ('music', 'Music'), ('concerts', 'Concerts'), ('movies', 'Movies'),
  ('series', 'TV series'), ('reading', 'Reading'), ('gaming', 'Gaming'), ('board-games', 'Board games'),
  ('fitness', 'Fitness'), ('running', 'Running'), ('yoga', 'Yoga'), ('cycling', 'Cycling'),
  ('climbing', 'Climbing'), ('swimming', 'Swimming'), ('football', 'Football'), ('dancing', 'Dancing'),
  ('photography', 'Photography'), ('art', 'Art'), ('museums', 'Museums'), ('theatre', 'Theatre'),
  ('animals', 'Animals'), ('gardening', 'Gardening'), ('tech', 'Tech'), ('startups', 'Startups'),
  ('volunteering', 'Volunteering'), ('languages', 'Languages'), ('food', 'Foodie'), ('beach', 'Beach'),
  ('camping', 'Camping'), ('skiing', 'Skiing'), ('podcasts', 'Podcasts');

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
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY IMMEDIATE
);

CREATE INDEX idx_photos_user_created_at ON photos (user_id, created_at);
