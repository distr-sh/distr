ALTER TABLE DeploymentResourceMetrics
  DROP COLUMN IF EXISTS log_bytes;

ALTER TABLE DeploymentTargetMetrics
  DROP COLUMN IF EXISTS agent_log_bytes,
  DROP COLUMN IF EXISTS image_bytes;
