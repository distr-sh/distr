ALTER TABLE DeploymentRevision
  DROP COLUMN IF EXISTS trigger;

DROP TYPE IF EXISTS DEPLOYMENT_REVISION_TRIGGER;

ALTER TABLE Deployment
  DROP COLUMN IF EXISTS automatic_application_updates_enabled;

ALTER TABLE Application
  DROP COLUMN IF EXISTS versioning_strategy,
  DROP COLUMN IF EXISTS allow_automatic_updates;

DROP TYPE IF EXISTS VERSIONING_STRATEGY;

-- An enum value cannot be removed again, so the feature is only taken off the organizations.
UPDATE Organization SET features = array_remove(features, 'auto_updates')
WHERE 'auto_updates' = ANY(features);
