-- Databases created by the previous SaaS boilerplate contain tables that the
-- dating product no longer uses. Drop them so the schema below starts clean.
-- On a fresh database every statement is a no-op.
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS votes CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS files CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS conversation_participants CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
    ALTER TABLE users DROP COLUMN IF EXISTS organization_id;
    ALTER TABLE users DROP COLUMN IF EXISTS display_name;
    ALTER TABLE users DROP COLUMN IF EXISTS bio;
  END IF;
END $$;

DROP TABLE IF EXISTS organizations CASCADE;
