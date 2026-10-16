ALTER TABLE DeploymentTarget
  DROP CONSTRAINT DeploymentTarget_access_key_complete,
  DROP CONSTRAINT DeploymentTarget_pending_access_key_complete,
  DROP COLUMN pending_access_key_salt,
  DROP COLUMN pending_access_key_hash;
