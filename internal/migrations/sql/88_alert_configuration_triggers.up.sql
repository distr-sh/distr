ALTER TABLE AlertConfiguration
  ADD COLUMN status_trigger_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN cpu_trigger_threshold_percent INT,
  ADD COLUMN memory_trigger_threshold_percent INT,
  ADD COLUMN disk_trigger_threshold_percent INT;
