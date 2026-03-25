ALTER TABLE NotificationRecord
  DROP COLUMN type,
  DROP COLUMN metric_type,
  DROP COLUMN disk_device,
  DROP COLUMN disk_path,
  DROP COLUMN previous_deployment_target_metrics_id,
  DROP COLUMN current_deployment_target_metrics_id;

DROP TYPE NOTIFICATION_RECORD_TYPE;

ALTER TABLE AlertConfiguration
  DROP COLUMN status_trigger_enabled,
  DROP COLUMN cpu_trigger_threshold_percent,
  DROP COLUMN memory_trigger_threshold_percent,
  DROP COLUMN disk_trigger_threshold_percent;
