ALTER TABLE DeploymentTarget
  ADD COLUMN pending_access_key_salt BYTEA,
  ADD COLUMN pending_access_key_hash BYTEA,
  ADD CONSTRAINT DeploymentTarget_access_key_complete CHECK (
    (access_key_salt IS NULL) = (access_key_hash IS NULL)
  ),
  ADD CONSTRAINT DeploymentTarget_pending_access_key_complete CHECK (
    (pending_access_key_salt IS NULL) = (pending_access_key_hash IS NULL)
  );
