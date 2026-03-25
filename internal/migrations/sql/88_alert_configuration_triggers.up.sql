ALTER TABLE AlertConfiguration
  ADD COLUMN status_trigger_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN cpu_trigger_threshold_percent INT,
  ADD COLUMN memory_trigger_threshold_percent INT,
  ADD COLUMN disk_trigger_threshold_percent INT;

ALTER TABLE NotificationRecord
  ADD COLUMN metric_type TEXT,
  ADD COLUMN disk_device TEXT,
  ADD COLUMN disk_path TEXT,
  ADD COLUMN previous_deployment_target_metrics_id UUID REFERENCES DeploymentTargetMetrics (id),
  ADD COLUMN current_deployment_target_metrics_id UUID REFERENCES DeploymentTargetMetrics (id);
