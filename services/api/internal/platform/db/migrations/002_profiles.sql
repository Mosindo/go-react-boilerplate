CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (char_length(first_name) BETWEEN 1 AND 50),
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('man', 'woman', 'non_binary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
  city TEXT NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
  -- Coordinates are stored rounded to 2 decimals (~1 km) and never returned to clients.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  location_updated_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX idx_profiles_geo ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;
CREATE INDEX idx_profiles_gender_birth ON profiles (gender, birth_date);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT ARRAY['man', 'woman', 'non_binary'],
  min_age INT NOT NULL DEFAULT 18 CHECK (min_age BETWEEN 18 AND 99),
  max_age INT NOT NULL DEFAULT 99 CHECK (max_age BETWEEN 18 AND 99),
  max_distance_km INT NOT NULL DEFAULT 50 CHECK (max_distance_km BETWEEN 1 AND 500),
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (min_age <= max_age),
  CONSTRAINT preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'non_binary']
  )
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
CREATE INDEX idx_user_interests_interest ON user_interests (interest_id, user_id);

-- Reference data (not demo users).
INSERT INTO interests (slug, label) VALUES
  ('hiking', 'Hiking'), ('cooking', 'Cooking'), ('travel', 'Travel'), ('music', 'Music'),
  ('concerts', 'Concerts'), ('movies', 'Movies'), ('reading', 'Reading'), ('gaming', 'Gaming'),
  ('fitness', 'Fitness'), ('yoga', 'Yoga'), ('running', 'Running'), ('cycling', 'Cycling'),
  ('photography', 'Photography'), ('art', 'Art'), ('museums', 'Museums'), ('theatre', 'Theatre'),
  ('dancing', 'Dancing'), ('coffee', 'Coffee'), ('wine', 'Wine'), ('foodie', 'Foodie'),
  ('pets', 'Pets'), ('gardening', 'Gardening'), ('tech', 'Tech'), ('writing', 'Writing'),
  ('board_games', 'Board games'), ('beach', 'Beach'), ('skiing', 'Skiing'), ('football', 'Football'),
  ('volunteering', 'Volunteering'), ('languages', 'Languages');
