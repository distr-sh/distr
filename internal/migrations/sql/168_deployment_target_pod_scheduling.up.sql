ALTER TABLE DeploymentTarget
  ADD COLUMN pod_annotations JSONB,
  ADD COLUMN node_selector JSONB,
  ADD COLUMN tolerations JSONB,
  ADD CONSTRAINT type_docker_pod_scheduling_check CHECK (
    type != 'docker'
    OR
    (pod_annotations IS NULL AND node_selector IS NULL AND tolerations IS NULL)
  );
