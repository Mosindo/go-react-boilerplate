CREATE TABLE IF NOT EXISTS comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  author_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  parent_comment_id UUID NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE comments
  ADD COLUMN IF NOT EXISTS parent_comment_id UUID NULL;

CREATE INDEX IF NOT EXISTS idx_comments_post_created_at
  ON comments (post_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_comments_author_created_at
  ON comments (author_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_comments_parent_created_at
  ON comments (parent_comment_id, created_at DESC);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_comments_parent_comment_id'
  ) THEN
    ALTER TABLE comments
      ADD CONSTRAINT fk_comments_parent_comment_id
      FOREIGN KEY (parent_comment_id)
      REFERENCES comments(id)
      ON DELETE CASCADE;
  END IF;
END $$;
