-- The SaaS-era modules (billing, posts, comments, votes, files, organizations)
-- no longer exist. Drop their tables and the legacy user columns.

DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS votes CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS files CASCADE;

ALTER TABLE users DROP COLUMN IF EXISTS organization_id;
ALTER TABLE users DROP COLUMN IF EXISTS display_name;
ALTER TABLE users DROP COLUMN IF EXISTS bio;

DROP TABLE IF EXISTS organizations CASCADE;
