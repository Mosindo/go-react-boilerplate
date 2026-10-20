CREATE TABLE IF NOT EXISTS posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  author_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'published',
  published_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE posts
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published';

ALTER TABLE posts
  ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ NULL;

ALTER TABLE posts
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_posts_author_created_at
  ON posts (author_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_posts_created_at
  ON posts (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_posts_status_created_at
  ON posts (status, created_at DESC);
