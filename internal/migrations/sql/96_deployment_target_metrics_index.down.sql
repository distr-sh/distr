DROP INDEX IF EXISTS DeploymentTargetMetrics_deployment_target_id_created_at_id;

CREATE INDEX IF NOT EXISTS DeploymentTargetMetrics_deployment_target_id ON DeploymentTargetMetrics (deployment_target_id);

-- Restore the misnamed index on DeploymentRevisionStatus (originally added by a typo in migration 37)
CREATE INDEX IF NOT EXISTS DeploymentTargetMetrics_created_at ON DeploymentRevisionStatus (created_at DESC);
