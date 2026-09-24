DROP INDEX fk_Deployment_latest_deployment_revision_id;
DROP INDEX fk_Deployment_current_deployment_revision_id;

ALTER TABLE Deployment
  DROP COLUMN latest_deployment_revision_id,
  DROP COLUMN current_deployment_revision_id;
