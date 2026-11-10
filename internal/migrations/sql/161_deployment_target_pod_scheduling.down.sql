ALTER TABLE DeploymentTarget
  DROP COLUMN pod_annotations,
  DROP COLUMN node_selector,
  DROP COLUMN tolerations;
