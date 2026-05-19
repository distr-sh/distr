ALTER TABLE Artifact
  DROP COLUMN IF EXISTS upstream_url,
  DROP COLUMN IF EXISTS last_synced_at,
  DROP COLUMN IF EXISTS last_sync_error,
  DROP COLUMN IF EXISTS upstream_auth_type,
  DROP COLUMN IF EXISTS upstream_username,
  DROP COLUMN IF EXISTS upstream_password;
