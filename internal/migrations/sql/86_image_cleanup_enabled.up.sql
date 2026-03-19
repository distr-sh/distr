ALTER TABLE DeploymentTarget
    ADD COLUMN image_cleanup_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD CONSTRAINT image_cleanup_enabled_only_for_docker
        CHECK (type = 'docker' OR NOT image_cleanup_enabled);
