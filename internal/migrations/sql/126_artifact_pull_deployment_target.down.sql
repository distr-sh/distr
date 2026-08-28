DROP INDEX fk_ArtifactVersionPull_deployment_target_id;

ALTER TABLE ArtifactVersionPull
  DROP COLUMN deployment_target_id;
