ALTER TABLE DeploymentTarget
  ADD COLUMN deployment_logs_enabled BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN deployment_logs_after TIMESTAMP;

UPDATE DeploymentTarget dt
SET deployment_logs_enabled = true
WHERE EXISTS (
  SELECT 1 FROM Deployment d WHERE d.deployment_target_id = dt.id AND d.logs_enabled = true
);

ALTER TABLE Deployment
  DROP COLUMN logs_enabled;
