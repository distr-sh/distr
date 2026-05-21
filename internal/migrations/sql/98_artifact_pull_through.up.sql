CREATE TYPE UPSTREAM_AUTH_TYPE AS ENUM ('basic', 'aws_ecr');

ALTER TABLE Artifact
  ADD COLUMN upstream_url TEXT,
  ADD COLUMN last_synced_at TIMESTAMP,
  ADD COLUMN last_sync_error TEXT,
  ADD COLUMN upstream_auth_type UPSTREAM_AUTH_TYPE,
  ADD COLUMN upstream_username TEXT,
  ADD COLUMN upstream_password TEXT;
