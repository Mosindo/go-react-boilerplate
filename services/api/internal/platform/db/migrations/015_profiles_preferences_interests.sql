CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  -- Coordinates are rounded to 2 decimals (~1 km) before storage: data minimisation.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  show_age BOOLEAN NOT NULL DEFAULT TRUE,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'non_binary')),
  CONSTRAINT chk_profiles_first_name CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_city CHECK (char_length(city) <= 80),
  CONSTRAINT chk_profiles_lat CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_lng CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180)
);

CREATE INDEX IF NOT EXISTS idx_profiles_discoverable_geo
  ON profiles (latitude, longitude) WHERE discoverable = TRUE;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  age_min SMALLINT NOT NULL DEFAULT 18,
  age_max SMALLINT NOT NULL DEFAULT 99,
  max_distance_km INTEGER NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (
    cardinality(interested_in) >= 1
    AND interested_in <@ ARRAY['woman', 'man', 'non_binary']::TEXT[]
  ),
  CONSTRAINT chk_preferences_age CHECK (age_min >= 18 AND age_max <= 99 AND age_min <= age_max),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO interests (slug, label) VALUES
  ('travel', 'Travel'), ('cooking', 'Cooking'), ('music', 'Music'), ('cinema', 'Cinema'),
  ('books', 'Books'), ('hiking', 'Hiking'), ('sports', 'Sports'), ('yoga', 'Yoga'),
  ('gaming', 'Gaming'), ('photography', 'Photography'), ('art', 'Art'), ('dancing', 'Dancing'),
  ('coffee', 'Coffee'), ('wine', 'Wine'), ('pets', 'Pets'), ('gardening', 'Gardening'),
  ('technology', 'Technology'), ('fitness', 'Fitness'), ('theatre', 'Theatre'), ('festivals', 'Festivals'),
  ('cycling', 'Cycling'), ('running', 'Running'), ('board_games', 'Board games'), ('volunteering', 'Volunteering'),
  ('languages', 'Languages'), ('fashion', 'Fashion'), ('food', 'Foodie'), ('nature', 'Nature'),
  ('meditation', 'Meditation'), ('podcasts', 'Podcasts')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest ON user_interests (interest_id, user_id);
