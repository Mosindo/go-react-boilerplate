-- Moderation: a suspended account cannot sign in and is removed from every surface.
ALTER TABLE users ADD COLUMN banned_at TIMESTAMPTZ NULL;
ALTER TABLE reports ADD COLUMN reviewed_at TIMESTAMPTZ NULL;
CREATE INDEX idx_users_banned ON users (banned_at) WHERE banned_at IS NOT NULL;
