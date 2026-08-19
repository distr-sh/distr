CREATE TABLE DeploymentMetrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  deployment_id UUID NOT NULL REFERENCES Deployment(id) ON DELETE CASCADE
);

-- Serves both the latest-report lookup and the age filter of the cleanup job. A BRIN index on
-- created_at was considered for cleanup and rejected: BRIN relies on physical row order
-- correlating with timestamp order, but the cleanup DELETEs plus vacuum space reuse destroy that
-- correlation on this table. If cleanup ever becomes a measured bottleneck, the escalation is
-- time-based partitioning, not BRIN.
CREATE INDEX IF NOT EXISTS DeploymentMetrics_deployment_id_created_at_id
  ON DeploymentMetrics (deployment_id, created_at DESC, id);

CREATE TABLE DeploymentWorkloadMetrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  deployment_metrics_id UUID NOT NULL REFERENCES DeploymentMetrics(id) ON DELETE CASCADE,
  workload TEXT NOT NULL,
  name TEXT NOT NULL,
  cpu_usage_millis BIGINT NOT NULL,
  memory_bytes BIGINT NOT NULL,
  -- NULL when the container (or any container of the pod) has no limit configured
  cpu_limit_millis BIGINT,
  memory_limit_bytes BIGINT
);

CREATE INDEX IF NOT EXISTS DeploymentWorkloadMetrics_metrics_id
  ON DeploymentWorkloadMetrics(deployment_metrics_id);

ALTER TABLE DeploymentTargetMetrics
  ADD COLUMN agent_cpu_usage_millis BIGINT,
  ADD COLUMN agent_memory_bytes BIGINT;
