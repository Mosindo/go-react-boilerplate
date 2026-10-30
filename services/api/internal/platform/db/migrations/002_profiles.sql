CREATE TABLE interests (
  id SMALLSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO interests (slug, label) VALUES
  ('voyage', 'Voyage'), ('cuisine', 'Cuisine'), ('cinema', 'Cinéma'), ('musique', 'Musique'),
  ('lecture', 'Lecture'), ('sport', 'Sport'), ('randonnee', 'Randonnée'), ('yoga', 'Yoga'),
  ('photographie', 'Photographie'), ('art', 'Art'), ('jeux-video', 'Jeux vidéo'), ('jeux-de-societe', 'Jeux de société'),
  ('theatre', 'Théâtre'), ('danse', 'Danse'), ('concerts', 'Concerts'), ('animaux', 'Animaux'),
  ('jardinage', 'Jardinage'), ('technologie', 'Technologie'), ('mode', 'Mode'), ('bricolage', 'Bricolage'),
  ('cafe', 'Café'), ('vin', 'Vin'), ('running', 'Course à pied'), ('velo', 'Vélo'),
  ('plage', 'Plage'), ('montagne', 'Montagne'), ('benevolat', 'Bénévolat'), ('ecriture', 'Écriture')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  display_name TEXT NOT NULL,
  birth_date DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  is_discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_name CHECK (char_length(display_name) BETWEEN 1 AND 50),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'non_binary')),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_city CHECK (char_length(city) <= 100),
  CONSTRAINT chk_profiles_lat CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_profiles_lon CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT chk_profiles_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL))
);

CREATE INDEX idx_profiles_discovery ON profiles (is_discoverable, gender) WHERE latitude IS NOT NULL;

CREATE TABLE preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  interested_in TEXT[] NOT NULL DEFAULT '{}',
  min_age SMALLINT NOT NULL DEFAULT 18,
  max_age SMALLINT NOT NULL DEFAULT 99,
  max_distance_km INTEGER NOT NULL DEFAULT 50,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_preferences_interested CHECK (interested_in <@ ARRAY['woman', 'man', 'non_binary']::TEXT[]),
  CONSTRAINT chk_preferences_age CHECK (min_age >= 18 AND max_age <= 99 AND min_age <= max_age),
  CONSTRAINT chk_preferences_distance CHECK (max_distance_km BETWEEN 1 AND 1000)
);

CREATE TABLE user_interests (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id SMALLINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX idx_user_interests_interest ON user_interests (interest_id, user_id);

CREATE TABLE photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  position SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE UNIQUE INDEX idx_photos_storage_key ON photos (storage_key);
