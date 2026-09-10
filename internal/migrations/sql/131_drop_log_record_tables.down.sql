CREATE TABLE IF NOT EXISTS DeploymentLogRecord (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP DEFAULT now(),
  deployment_id UUID NOT NULL REFERENCES Deployment (id) ON DELETE CASCADE,
  deployment_revision_id UUID NOT NULL REFERENCES DeploymentRevision (id) ON DELETE CASCADE,
  resource TEXT,
  timestamp TIMESTAMP,
  severity TEXT,
  body TEXT
);

CREATE INDEX IF NOT EXISTS fk_DeploymentLogRecord_deployment_id ON DeploymentLogRecord (deployment_id);
CREATE INDEX IF NOT EXISTS fk_DeploymentLogRecord_deployment_revision_id
  ON DeploymentLogRecord (deployment_revision_id);
CREATE INDEX IF NOT EXISTS DeploymentLogRecord_resource ON DeploymentLogRecord (resource);
CREATE INDEX IF NOT EXISTS DeploymentLogRecord_deployment_id_resource_timestamp
  ON DeploymentLogRecord (deployment_id, resource, timestamp DESC);
CREATE INDEX IF NOT EXISTS DeploymentLogRecord_deployment_revision_id_resource
  ON DeploymentLogRecord (deployment_revision_id, resource);

CREATE TABLE IF NOT EXISTS DeploymentTargetLogRecord (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  deployment_target_id UUID NOT NULL REFERENCES DeploymentTarget (id) ON DELETE CASCADE,
  timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
  severity TEXT NOT NULL,
  body TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS DeploymentTargetLogRecord_deployment_target_id_timestamp
  ON DeploymentTargetLogRecord (deployment_target_id, timestamp DESC);
