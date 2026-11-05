DROP INDEX fk_AlertConfiguration_customer_organization_id;

CREATE INDEX fk_ArtifactVersionPull_customer_organization_id ON ArtifactVersionPull (customer_organization_id);
DROP INDEX idx_artifactversionpull_customer_organization_id_created_at;

CREATE INDEX fk_DeploymentRevision_deployment_id ON DeploymentRevision (deployment_id);
DROP INDEX idx_deploymentrevision_deployment_id_created_at;

ALTER TABLE Artifact DROP COLUMN featured;
ALTER TABLE Artifact DROP COLUMN title_image_id;
ALTER TABLE Application DROP COLUMN title_image_id;
