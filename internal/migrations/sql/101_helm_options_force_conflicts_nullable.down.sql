UPDATE DeploymentRevision SET helm_options_force_conflicts = FALSE WHERE helm_options_force_conflicts IS NULL;
ALTER TABLE DeploymentRevision
  ALTER COLUMN helm_options_force_conflicts SET DEFAULT FALSE,
  ALTER COLUMN helm_options_force_conflicts SET NOT NULL;
