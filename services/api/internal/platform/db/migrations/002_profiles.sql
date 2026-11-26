CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  -- Coarse coordinates only (rounded server-side to ~1 km). Exact GPS positions are never stored.
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_first_name CHECK (length(btrim(first_name)) BETWEEN 1 AND 40),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_bio CHECK (length(bio) <= 500),
  CONSTRAINT chk_profiles_job_title CHECK (length(job_title) <= 60),
  CONSTRAINT chk_profiles_city CHECK (length(city) <= 60),
  CONSTRAINT chk_profiles_location CHECK (
    (latitude IS NULL AND longitude IS NULL)
    OR (latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180)
  )
);

CREATE INDEX idx_profiles_discovery ON profiles (gender, birthdate) WHERE discoverable;
CREATE INDEX idx_profiles_location ON profiles (latitude, longitude) WHERE latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL,
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km INTEGER NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested_in CHECK (
    cardinality(interested_in) BETWEEN 1 AND 3
    AND interested_in <@ ARRAY['woman', 'man', 'nonbinary']::TEXT[]
  ),
  CONSTRAINT chk_preferences_age CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 500)
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

CREATE INDEX idx_user_interests_interest_id ON user_interests (interest_id, user_id);

INSERT INTO interests (slug, label, category) VALUES
  ('hiking', 'Randonnée', 'Plein air'),
  ('climbing', 'Escalade', 'Plein air'),
  ('cycling', 'Vélo', 'Plein air'),
  ('running', 'Course à pied', 'Sport'),
  ('yoga', 'Yoga', 'Sport'),
  ('swimming', 'Natation', 'Sport'),
  ('football', 'Football', 'Sport'),
  ('tennis', 'Tennis', 'Sport'),
  ('dance', 'Danse', 'Sport'),
  ('gym', 'Musculation', 'Sport'),
  ('cooking', 'Cuisine', 'Gastronomie'),
  ('wine', 'Vin', 'Gastronomie'),
  ('coffee', 'Café', 'Gastronomie'),
  ('brunch', 'Brunch', 'Gastronomie'),
  ('vegetarian', 'Cuisine végétarienne', 'Gastronomie'),
  ('travel', 'Voyages', 'Découverte'),
  ('roadtrips', 'Road trips', 'Découverte'),
  ('museums', 'Musées', 'Culture'),
  ('theatre', 'Théâtre', 'Culture'),
  ('cinema', 'Cinéma', 'Culture'),
  ('reading', 'Lecture', 'Culture'),
  ('writing', 'Écriture', 'Culture'),
  ('photography', 'Photographie', 'Créativité'),
  ('drawing', 'Dessin', 'Créativité'),
  ('diy', 'Bricolage', 'Créativité'),
  ('gardening', 'Jardinage', 'Créativité'),
  ('concerts', 'Concerts', 'Musique'),
  ('festivals', 'Festivals', 'Musique'),
  ('playing_music', 'Jouer de la musique', 'Musique'),
  ('jazz', 'Jazz', 'Musique'),
  ('electro', 'Électro', 'Musique'),
  ('video_games', 'Jeux vidéo', 'Loisirs'),
  ('board_games', 'Jeux de société', 'Loisirs'),
  ('podcasts', 'Podcasts', 'Loisirs'),
  ('series', 'Séries', 'Loisirs'),
  ('animals', 'Animaux', 'Vie'),
  ('volunteering', 'Bénévolat', 'Vie'),
  ('meditation', 'Méditation', 'Vie'),
  ('tech', 'Tech', 'Vie'),
  ('languages', 'Langues étrangères', 'Vie');
