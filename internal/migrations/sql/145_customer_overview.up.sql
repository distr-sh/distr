ALTER TABLE Application ADD COLUMN title_image_id UUID REFERENCES File (id) ON DELETE SET NULL;
ALTER TABLE Artifact ADD COLUMN title_image_id UUID REFERENCES File (id) ON DELETE SET NULL;
ALTER TABLE Artifact ADD COLUMN hidden_in_customer_overview BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX fk_Application_title_image_id ON Application (title_image_id) WHERE title_image_id IS NOT NULL;
CREATE INDEX fk_Artifact_title_image_id ON Artifact (title_image_id) WHERE title_image_id IS NOT NULL;

CREATE INDEX idx_deploymentrevision_deployment_id_created_at
  ON DeploymentRevision (deployment_id, created_at DESC);
DROP INDEX fk_DeploymentRevision_deployment_id;

CREATE INDEX idx_applicationversion_application_id_created_at
  ON ApplicationVersion (application_id, created_at DESC);
DROP INDEX fk_ApplicationVersion_application_id;

CREATE INDEX idx_artifactversion_artifact_id_created_at
  ON ArtifactVersion (artifact_id, created_at DESC);
DROP INDEX fk_ArtifactVersion_artifact_id;

CREATE INDEX idx_artifactversionpull_version_user_created_at
  ON ArtifactVersionPull (artifact_version_id, useraccount_id, created_at);
DROP INDEX fk_ArtifactVersionPull_artifact_version_id;

CREATE INDEX fk_ArtifactVersionPull_useraccount_id ON ArtifactVersionPull (useraccount_id);

CREATE INDEX idx_artifactversionpull_customer_organization_id_created_at
  ON ArtifactVersionPull (customer_organization_id, created_at DESC);
DROP INDEX fk_ArtifactVersionPull_customer_organization_id;

CREATE INDEX fk_AlertConfiguration_customer_organization_id ON AlertConfiguration (customer_organization_id);
