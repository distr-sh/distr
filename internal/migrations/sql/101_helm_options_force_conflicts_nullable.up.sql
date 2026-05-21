ALTER TABLE DeploymentRevision
  ALTER COLUMN helm_options_force_conflicts DROP NOT NULL,
  ALTER COLUMN helm_options_force_conflicts DROP DEFAULT;
