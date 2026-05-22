UPDATE DeploymentRevision SET helm_options_force_conflicts = NULL WHERE helm_options_timeout IS NULL;

ALTER TABLE DeploymentRevision
  ADD CONSTRAINT helm_options_all_or_none CHECK (
    num_nonnulls(
      helm_options_timeout,
      helm_options_wait_strategy,
      helm_options_rollback_on_failure,
      helm_options_cleanup_on_failure,
      helm_options_force_conflicts
    ) IN (0, 5)
  );
