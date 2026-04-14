ALTER TABLE DeploymentTarget
  ADD COLUMN logs_enabled BOOLEAN NOT NULL DEFAULT false;

UPDATE DeploymentTarget dt
SET logs_enabled = true
WHERE EXISTS (
  SELECT 1 FROM Deployment d WHERE d.deployment_target_id = dt.id AND d.logs_enabled = true
);

ALTER TABLE Deployment
  DROP COLUMN logs_enabled;
