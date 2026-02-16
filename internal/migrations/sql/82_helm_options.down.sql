ALTER TABLE DeploymentRevision
  DROP helm_options_timeout,
  DROP helm_options_wait_strategy,
  DROP helm_options_rollback_on_failure,
  DROP helm_options_cleanup_on_failure;
