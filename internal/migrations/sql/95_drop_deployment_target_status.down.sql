CREATE TABLE IF NOT EXISTS DeploymentTargetStatus (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP DEFAULT current_timestamp,
  deployment_target_id UUID NOT NULL REFERENCES DeploymentTarget (id) ON DELETE CASCADE,
  message TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS fk_DeploymentTargetStatus_deployment_target_id ON DeploymentTargetStatus (deployment_target_id);
CREATE INDEX DeploymentTargetStatus_created_at ON DeploymentTargetStatus (deployment_target_id, created_at DESC);
