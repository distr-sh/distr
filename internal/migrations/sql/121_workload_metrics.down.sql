ALTER TABLE DeploymentTargetMetrics
  DROP COLUMN IF EXISTS agent_cpu_usage_millis,
  DROP COLUMN IF EXISTS agent_memory_bytes;

DROP TABLE IF EXISTS DeploymentWorkloadMetrics;

DROP TABLE IF EXISTS DeploymentMetrics;
