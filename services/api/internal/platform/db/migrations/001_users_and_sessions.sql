CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  -- Demo accounts created by `go run ./cmd/seed`. They can never log in
  -- (random password) and are excluded from discovery in production.
  is_demo BOOLEAN NOT NULL DEFAULT FALSE,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_users_email_normalized CHECK (email = lower(btrim(email)) AND length(email) BETWEEN 3 AND 254)
);

CREATE UNIQUE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_demo ON users (is_demo) WHERE is_demo;

CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  user_agent TEXT NULL,
  ip_address INET NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ NULL,
  last_used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions (token_hash);
CREATE INDEX idx_sessions_user_created_at ON sessions (user_id, created_at DESC);

CREATE TABLE password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_password_reset_tokens_hash ON password_reset_tokens (token_hash);
CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens (user_id, created_at DESC);
