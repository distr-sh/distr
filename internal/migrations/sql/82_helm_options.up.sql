ALTER TABLE DeploymentRevision
  ADD COLUMN helm_options_timeout TEXT,
  ADD COLUMN helm_options_wait_strategy TEXT,
  ADD COLUMN helm_options_rollback_on_failure BOOLEAN,
  ADD COLUMN helm_options_cleanup_on_failure BOOLEAN;
