ALTER TABLE DeploymentTarget
    ADD COLUMN autoheal_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE DeploymentTarget
    ADD CONSTRAINT deployment_target_autoheal_enabled_check
        CHECK (NOT (autoheal_enabled = TRUE AND type = 'kubernetes'));
