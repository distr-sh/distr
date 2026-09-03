ALTER TABLE ArtifactVersionPull
  ADD COLUMN deployment_target_id UUID REFERENCES DeploymentTarget (id) ON DELETE SET NULL;

CREATE INDEX fk_ArtifactVersionPull_deployment_target_id ON ArtifactVersionPull (deployment_target_id);
