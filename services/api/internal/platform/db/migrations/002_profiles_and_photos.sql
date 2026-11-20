CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  birthdate DATE NOT NULL,
  gender TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  job_title TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  height_cm SMALLINT NULL,
  relationship_goal TEXT NULL,
  -- Coordinates are snapped to a ~1 km grid before storage; exact GPS positions are never persisted.
  location_lat DOUBLE PRECISION NULL,
  location_lng DOUBLE PRECISION NULL,
  location_updated_at TIMESTAMPTZ NULL,
  discoverable BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_profiles_first_name CHECK (char_length(first_name) BETWEEN 1 AND 40),
  CONSTRAINT chk_profiles_gender CHECK (gender IN ('woman', 'man', 'nonbinary')),
  CONSTRAINT chk_profiles_bio CHECK (char_length(bio) <= 500),
  CONSTRAINT chk_profiles_job_title CHECK (char_length(job_title) <= 60),
  CONSTRAINT chk_profiles_city CHECK (char_length(city) <= 60),
  CONSTRAINT chk_profiles_height CHECK (height_cm IS NULL OR height_cm BETWEEN 120 AND 230),
  CONSTRAINT chk_profiles_relationship_goal CHECK (relationship_goal IS NULL OR relationship_goal IN ('long_term', 'short_term', 'friendship', 'undecided')),
  CONSTRAINT chk_profiles_adult CHECK (birthdate <= (CURRENT_DATE - INTERVAL '18 years')),
  CONSTRAINT chk_profiles_location_pair CHECK ((location_lat IS NULL) = (location_lng IS NULL)),
  CONSTRAINT chk_profiles_location_range CHECK (
    location_lat IS NULL OR (location_lat BETWEEN -90 AND 90 AND location_lng BETWEEN -180 AND 180)
  )
);

CREATE INDEX idx_profiles_discovery ON profiles (gender, birthdate) WHERE discoverable;
CREATE INDEX idx_profiles_location ON profiles (location_lat, location_lng) WHERE location_lat IS NOT NULL;

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
  label TEXT NOT NULL,
  category TEXT NOT NULL
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
  position SMALLINT NOT NULL,
  storage_key TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_photos_position CHECK (position BETWEEN 0 AND 5),
  CONSTRAINT chk_photos_mime CHECK (mime_type IN ('image/jpeg')),
  CONSTRAINT chk_photos_dimensions CHECK (width > 0 AND height > 0 AND size_bytes > 0),
  -- Deferred so photos can be reordered inside one transaction.
  CONSTRAINT uq_photos_user_position UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE UNIQUE INDEX idx_photos_storage_key ON photos (storage_key);
