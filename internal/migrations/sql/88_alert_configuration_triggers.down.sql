ALTER TABLE AlertConfiguration
  DROP COLUMN status_trigger_enabled,
  DROP COLUMN cpu_trigger_threshold_percent,
  DROP COLUMN memory_trigger_threshold_percent,
  DROP COLUMN disk_trigger_threshold_percent;
