ALTER TABLE DeploymentTarget
    ADD COLUMN docker_endpoint TEXT,
    ADD CONSTRAINT deployment_target_docker_endpoint_check
        CHECK (type = 'docker' OR docker_endpoint IS NULL);
