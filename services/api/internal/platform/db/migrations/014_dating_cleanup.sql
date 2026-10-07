-- The product is a free dating app: the generic social/SaaS tables from the
-- original boilerplate (posts, comments, votes, files, organizations,
-- subscriptions and the first chat/notification schemas) are retired here.
-- Dating tables are created in the following migrations.
DROP TABLE IF EXISTS votes, comments, posts, files, subscriptions CASCADE;
DROP TABLE IF EXISTS messages, conversation_participants, conversations CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;

ALTER TABLE users DROP COLUMN IF EXISTS organization_id;
ALTER TABLE users DROP COLUMN IF EXISTS display_name;
ALTER TABLE users DROP COLUMN IF EXISTS bio;
DROP TABLE IF EXISTS organizations CASCADE;

ALTER TABLE users ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_email_lower;
ALTER TABLE users ADD CONSTRAINT chk_users_email_lower CHECK (email = lower(email)) NOT VALID;
