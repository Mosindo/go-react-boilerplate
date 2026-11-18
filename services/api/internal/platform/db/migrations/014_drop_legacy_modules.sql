-- The product is a free dating app: billing, public posts/comments/votes and the
-- metadata-only files stub are no longer part of the platform.
DROP TABLE IF EXISTS votes CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS files CASCADE;
