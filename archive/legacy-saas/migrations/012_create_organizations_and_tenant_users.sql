CREATE TABLE IF NOT EXISTS organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO organizations (slug, name)
VALUES ('default', 'Default Workspace')
ON CONFLICT (slug) DO NOTHING;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS organization_id UUID NULL;

UPDATE users
SET organization_id = (SELECT id FROM organizations WHERE slug = 'default')
WHERE organization_id IS NULL;

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_organization_id_fkey;

ALTER TABLE users
  ADD CONSTRAINT users_organization_id_fkey
  FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE RESTRICT;

ALTER TABLE users
  ALTER COLUMN organization_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_organization_id_created_at
  ON users (organization_id, created_at DESC, id DESC);
