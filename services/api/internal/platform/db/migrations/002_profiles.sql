-- Location is stored rounded to 2 decimals (~1 km): never an exact position.
CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_first_name_len CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT profiles_gender_valid CHECK (gender IN ('man', 'woman', 'non_binary')),
  CONSTRAINT profiles_bio_len CHECK (char_length(bio) <= 500),
  CONSTRAINT profiles_city_len CHECK (char_length(city) <= 80),
  CONSTRAINT profiles_lat_range CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT profiles_lng_range CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX idx_profiles_discovery ON profiles (discoverable, gender, birth_date);

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  age_min SMALLINT NOT NULL DEFAULT 18,
  age_max SMALLINT NOT NULL DEFAULT 99,
  max_distance_km INTEGER NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT preferences_age_range CHECK (age_min >= 18 AND age_max <= 99 AND age_min <= age_max),
  CONSTRAINT preferences_distance CHECK (max_distance_km BETWEEN 1 AND 20000),
  CONSTRAINT preferences_interested_valid CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['man', 'woman', 'non_binary']::text[]
  )
);

CREATE TABLE interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

-- Reference data (not demo data): the catalogue of selectable interests.
INSERT INTO interests (slug, label) VALUES
  ('travel', 'Travel'), ('cooking', 'Cooking'), ('music', 'Music'), ('movies', 'Movies'),
  ('books', 'Books'), ('hiking', 'Hiking'), ('fitness', 'Fitness'), ('yoga', 'Yoga'),
  ('photography', 'Photography'), ('art', 'Art'), ('gaming', 'Gaming'), ('dancing', 'Dancing'),
  ('coffee', 'Coffee'), ('wine', 'Wine'), ('animals', 'Animals'), ('cycling', 'Cycling'),
  ('running', 'Running'), ('theatre', 'Theatre'), ('tech', 'Tech'), ('nature', 'Nature'),
  ('festivals', 'Festivals'), ('board_games', 'Board games'), ('languages', 'Languages'), ('volunteering', 'Volunteering');

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
  position SMALLINT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT photos_position_range CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT photos_user_position_unique UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);
