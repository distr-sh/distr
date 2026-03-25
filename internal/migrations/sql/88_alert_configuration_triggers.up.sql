ALTER TABLE AlertConfiguration
  ADD COLUMN status_trigger_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN cpu_trigger_threshold_percent INT,
  ADD COLUMN memory_trigger_threshold_percent INT,
  ADD COLUMN disk_trigger_threshold_percent INT;

CREATE TYPE NOTIFICATION_RECORD_TYPE AS ENUM ('alert', 'warning', 'resolved');

ALTER TABLE NotificationRecord
  ADD COLUMN type NOTIFICATION_RECORD_TYPE,
  ADD COLUMN metric_type TEXT,
  ADD COLUMN disk_device TEXT,
  ADD COLUMN disk_path TEXT,
  ADD COLUMN previous_deployment_target_metrics_id UUID REFERENCES DeploymentTargetMetrics (id) ON DELETE CASCADE,
  ADD COLUMN current_deployment_target_metrics_id UUID REFERENCES DeploymentTargetMetrics (id) ON DELETE CASCADE;

UPDATE NotificationRecord
SET type = CASE
  WHEN (SELECT type FROM DeploymentRevisionStatus WHERE id = current_deployment_revision_status_id) = 'error' THEN 'alert'
  WHEN current_deployment_revision_status_id IS NOT NULL THEN 'resolved'
  ELSE 'warning'
END::NOTIFICATION_RECORD_TYPE;

ALTER TABLE NotificationRecord
  ALTER COLUMN type SET NOT NULL;
