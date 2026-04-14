ALTER TABLE Deployment
  ADD COLUMN logs_enabled BOOLEAN NOT NULL DEFAULT false;

UPDATE Deployment d
SET logs_enabled = true
WHERE EXISTS (
  SELECT 1 FROM DeploymentTarget dt WHERE dt.id = d.deployment_target_id AND dt.logs_enabled = true
);

ALTER TABLE DeploymentTarget
  DROP COLUMN logs_enabled;
