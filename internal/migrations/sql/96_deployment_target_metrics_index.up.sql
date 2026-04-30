-- DeploymentTargetMetrics_created_at was added by a typo in migration 37 and targets DeploymentRevisionStatus instead of DeploymentTargetMetrics
DROP INDEX IF EXISTS DeploymentTargetMetrics_created_at;

CREATE INDEX IF NOT EXISTS DeploymentTargetMetrics_deployment_target_id_created_at_id
    ON DeploymentTargetMetrics (deployment_target_id, created_at DESC, id);
