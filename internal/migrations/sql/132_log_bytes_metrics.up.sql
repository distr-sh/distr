ALTER TABLE DeploymentResourceMetrics
  -- NULL when the container uses a log driver that does not write files, and in kubernetes
  ADD COLUMN log_bytes BIGINT;

ALTER TABLE DeploymentTargetMetrics
  ADD COLUMN agent_log_bytes BIGINT;
