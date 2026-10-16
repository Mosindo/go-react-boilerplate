-- Limits how fast a profile location can be moved (trilateration defence) and keeps
-- the last messages of the conversation with a report, since blocking deletes the match.
ALTER TABLE profiles ADD COLUMN location_changed_at TIMESTAMPTZ NULL;
ALTER TABLE reports ADD COLUMN context JSONB NOT NULL DEFAULT '[]'::jsonb;
