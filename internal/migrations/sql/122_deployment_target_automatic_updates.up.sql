ALTER TABLE DeploymentTarget
    ADD COLUMN automatic_updates_enabled BOOLEAN NOT NULL DEFAULT false;
