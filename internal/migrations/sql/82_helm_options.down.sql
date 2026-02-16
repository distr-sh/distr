ALTER TABLE DeploymentRevision
  DROP COLUMN helm_options_timeout,
  DROP COLUMN helm_options_wait_strategy,
  DROP COLUMN helm_options_rollback_on_failure,
  DROP COLUMN helm_options_cleanup_on_failure;
