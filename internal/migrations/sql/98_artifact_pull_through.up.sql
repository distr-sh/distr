ALTER TABLE Artifact
  ADD COLUMN upstream_url TEXT,
  ADD COLUMN last_synced_at TIMESTAMP,
  ADD COLUMN last_sync_error TEXT,
  ADD COLUMN upstream_auth_type TEXT,
  ADD COLUMN upstream_username TEXT,
  ADD COLUMN upstream_password TEXT;
