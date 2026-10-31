ALTER TABLE Application ADD COLUMN title_image_id UUID REFERENCES File (id) ON DELETE SET NULL;
ALTER TABLE Artifact ADD COLUMN title_image_id UUID REFERENCES File (id) ON DELETE SET NULL;
ALTER TABLE Artifact ADD COLUMN featured BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS fk_Application_title_image_id ON Application (title_image_id) WHERE title_image_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS fk_Artifact_title_image_id ON Artifact (title_image_id) WHERE title_image_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_deploymentrevision_deployment_id_created_at
  ON DeploymentRevision (deployment_id, created_at DESC);
DROP INDEX fk_DeploymentRevision_deployment_id;

-- Building an index blocks writes to ArtifactVersionPull, and with them every registry pull, until it is done,
-- which takes a while on a large table. CONCURRENTLY avoids that but cannot run inside a migration, so on a large
-- installation create this index before upgrading and this migration skips it:
--
--   CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_artifactversionpull_customer_organization_id_created_at
--     ON ArtifactVersionPull (customer_organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifactversionpull_customer_organization_id_created_at
  ON ArtifactVersionPull (customer_organization_id, created_at DESC);
DROP INDEX fk_ArtifactVersionPull_customer_organization_id;

CREATE INDEX IF NOT EXISTS fk_AlertConfiguration_customer_organization_id
  ON AlertConfiguration (customer_organization_id);
