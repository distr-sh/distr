ALTER TABLE Deployment
  ADD COLUMN latest_deployment_revision_id UUID REFERENCES DeploymentRevision (id) ON DELETE SET NULL,
  ADD COLUMN current_deployment_revision_id UUID REFERENCES DeploymentRevision (id) ON DELETE SET NULL;

CREATE INDEX fk_Deployment_latest_deployment_revision_id ON Deployment (latest_deployment_revision_id)
  WHERE latest_deployment_revision_id IS NOT NULL;
CREATE INDEX fk_Deployment_current_deployment_revision_id ON Deployment (current_deployment_revision_id)
  WHERE current_deployment_revision_id IS NOT NULL;

UPDATE Deployment d SET latest_deployment_revision_id = (
  SELECT dr.id FROM DeploymentRevision dr
  WHERE dr.deployment_id = d.id
  ORDER BY dr.created_at DESC
  LIMIT 1
);

-- The current revision is the newest one an agent has reported as applied. Statuses older than
-- STATUS_ENTRIES_MAX_AGE are already deleted, so a deployment whose last applied status fell out of
-- that window keeps a NULL current revision until its agent sends the next status.
UPDATE Deployment d SET current_deployment_revision_id = (
  SELECT dr.id FROM DeploymentRevision dr
  WHERE dr.deployment_id = d.id
    AND EXISTS (
      SELECT 1 FROM DeploymentRevisionStatus drs
      WHERE drs.deployment_revision_id = dr.id AND drs.type IN ('healthy', 'running')
    )
  ORDER BY dr.created_at DESC
  LIMIT 1
);
