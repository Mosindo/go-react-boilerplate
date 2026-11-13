CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL DEFAULT '',
  birthdate DATE NULL,
  gender TEXT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  height_cm SMALLINT NULL,
  relationship_goal TEXT NULL,
  city TEXT NOT NULL DEFAULT '',
  -- Coordinates are rounded to 2 decimals (~1 km) before being stored.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  is_paused BOOLEAN NOT NULL DEFAULT FALSE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  moderation_hidden BOOLEAN NOT NULL DEFAULT FALSE,
  is_complete BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_first_name CHECK (char_length(first_name) <= 40),
  CONSTRAINT chk_profiles_gender CHECK (gender IS NULL OR gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_job_title CHECK (char_length(job_title) <= 60),
  CONSTRAINT chk_profiles_height CHECK (height_cm IS NULL OR height_cm BETWEEN 120 AND 230),
  CONSTRAINT chk_profiles_relationship_goal CHECK (
    relationship_goal IS NULL OR relationship_goal IN ('long_term', 'short_term', 'friendship', 'unsure')
  ),
  CONSTRAINT chk_profiles_city CHECK (char_length(city) <= 80),
  CONSTRAINT chk_profiles_latitude CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_longitude CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180)
);

-- Discovery prefilter: only complete, visible profiles, then a latitude band.
CREATE INDEX IF NOT EXISTS idx_profiles_discoverable_lat
  ON profiles (latitude, last_active_at DESC)
  WHERE is_complete AND NOT is_paused AND NOT moderation_hidden;

CREATE TABLE IF NOT EXISTS preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT '{}',
  age_min SMALLINT NOT NULL DEFAULT 18,
  age_max SMALLINT NOT NULL DEFAULT 99,
  max_distance_km SMALLINT NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]),
  CONSTRAINT chk_preferences_age CHECK (age_min >= 18 AND age_max <= 99 AND age_min <= age_max),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
);

CREATE TABLE IF NOT EXISTS interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  category TEXT NOT NULL
);

INSERT INTO interests (slug, label, category) VALUES
  ('hiking', 'Randonnée', 'outdoors'),
  ('cycling', 'Vélo', 'outdoors'),
  ('climbing', 'Escalade', 'outdoors'),
  ('running', 'Course à pied', 'sport'),
  ('yoga', 'Yoga', 'sport'),
  ('swimming', 'Natation', 'sport'),
  ('football', 'Football', 'sport'),
  ('tennis', 'Tennis', 'sport'),
  ('dance', 'Danse', 'sport'),
  ('cooking', 'Cuisine', 'food'),
  ('baking', 'Pâtisserie', 'food'),
  ('wine', 'Vin', 'food'),
  ('coffee', 'Café', 'food'),
  ('brunch', 'Brunch', 'food'),
  ('vegan', 'Cuisine végétale', 'food'),
  ('reading', 'Lecture', 'culture'),
  ('writing', 'Écriture', 'culture'),
  ('cinema', 'Cinéma', 'culture'),
  ('theatre', 'Théâtre', 'culture'),
  ('museums', 'Musées', 'culture'),
  ('photography', 'Photographie', 'culture'),
  ('painting', 'Peinture', 'culture'),
  ('concerts', 'Concerts', 'music'),
  ('jazz', 'Jazz', 'music'),
  ('electro', 'Électro', 'music'),
  ('rock', 'Rock', 'music'),
  ('hiphop', 'Hip-hop', 'music'),
  ('classical', 'Classique', 'music'),
  ('playing_music', 'Jouer de la musique', 'music'),
  ('travel', 'Voyages', 'lifestyle'),
  ('road_trips', 'Road trips', 'lifestyle'),
  ('gardening', 'Jardinage', 'lifestyle'),
  ('pets', 'Animaux', 'lifestyle'),
  ('volunteering', 'Bénévolat', 'lifestyle'),
  ('meditation', 'Méditation', 'lifestyle'),
  ('fashion', 'Mode', 'lifestyle'),
  ('board_games', 'Jeux de société', 'games'),
  ('video_games', 'Jeux vidéo', 'games'),
  ('podcasts', 'Podcasts', 'culture'),
  ('tech', 'Tech', 'culture'),
  ('science', 'Sciences', 'culture'),
  ('languages', 'Langues', 'culture')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest ON user_interests (interest_id, user_id);
