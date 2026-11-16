CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL CHECK (length(first_name) BETWEEN 1 AND 40),
  -- Birthdate is private; only the derived age is ever exposed.
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('woman', 'man', 'nonbinary')),
  bio TEXT NOT NULL DEFAULT '' CHECK (length(bio) <= 500),
  job_title TEXT NOT NULL DEFAULT '' CHECK (length(job_title) <= 60),
  height_cm SMALLINT NULL CHECK (height_cm BETWEEN 120 AND 230),
  looking_for TEXT NOT NULL DEFAULT 'unsure' CHECK (looking_for IN ('relationship', 'casual', 'friendship', 'unsure')),
  city TEXT NOT NULL DEFAULT '' CHECK (length(city) <= 80),
  -- Coordinates are rounded server-side (~1 km grid) before storage and are
  -- never returned to other users.
  latitude DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
  location_updated_at TIMESTAMPTZ NULL,
  is_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX idx_profiles_discovery ON profiles (gender, birthdate) WHERE is_discoverable;
CREATE INDEX idx_profiles_location ON profiles (latitude, longitude) WHERE is_discoverable AND latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  age_min SMALLINT NOT NULL DEFAULT 18,
  age_max SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  ),
  CONSTRAINT chk_preferences_age CHECK (age_min >= 18 AND age_max <= 99 AND age_min <= age_max),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
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

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  width INTEGER NOT NULL CHECK (width > 0),
  height INTEGER NOT NULL CHECK (height > 0),
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  position SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- Reference data (not demo data): the catalogue of selectable interests.
INSERT INTO interests (slug, label) VALUES
  ('hiking', 'Randonnée'),
  ('cooking', 'Cuisine'),
  ('travel', 'Voyages'),
  ('music', 'Musique'),
  ('concerts', 'Concerts'),
  ('cinema', 'Cinéma'),
  ('series', 'Séries'),
  ('reading', 'Lecture'),
  ('writing', 'Écriture'),
  ('photography', 'Photographie'),
  ('art', 'Art & musées'),
  ('theatre', 'Théâtre'),
  ('dance', 'Danse'),
  ('yoga', 'Yoga'),
  ('running', 'Course à pied'),
  ('cycling', 'Vélo'),
  ('climbing', 'Escalade'),
  ('swimming', 'Natation'),
  ('football', 'Football'),
  ('fitness', 'Fitness'),
  ('gaming', 'Jeux vidéo'),
  ('board_games', 'Jeux de société'),
  ('tech', 'Tech'),
  ('science', 'Sciences'),
  ('nature', 'Nature'),
  ('animals', 'Animaux'),
  ('gardening', 'Jardinage'),
  ('wine', 'Vin'),
  ('coffee', 'Café'),
  ('brunch', 'Brunch'),
  ('volunteering', 'Bénévolat'),
  ('languages', 'Langues'),
  ('fashion', 'Mode'),
  ('meditation', 'Méditation'),
  ('podcasts', 'Podcasts'),
  ('diy', 'DIY');
