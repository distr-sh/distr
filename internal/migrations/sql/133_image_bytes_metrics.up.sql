ALTER TABLE DeploymentTargetMetrics
  -- NULL in kubernetes, and in docker until the first collection has completed
  ADD COLUMN image_bytes BIGINT;
