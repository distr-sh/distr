CREATE TABLE DeploymentTargetDiskMetrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  deployment_target_metrics_id UUID NOT NULL REFERENCES DeploymentTargetMetrics(id) ON DELETE CASCADE,
  device TEXT NOT NULL,
  path TEXT NOT NULL,
  fs_type TEXT NOT NULL,
  bytes_total BIGINT NOT NULL,
  bytes_used BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS DeploymentTargetDiskMetrics_metrics_id
  ON DeploymentTargetDiskMetrics(deployment_target_metrics_id);
