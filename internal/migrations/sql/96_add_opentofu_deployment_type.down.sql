-- Keep opentofu in constraints since the enum value cannot be removed
ALTER TABLE DeploymentTarget
    DROP CONSTRAINT IF EXISTS namespace_required,
    ADD CONSTRAINT namespace_required CHECK (
        (type IN ('docker'::deployment_type, 'opentofu'::deployment_type)) = (namespace IS NULL)
    );

ALTER TABLE DeploymentTarget
    DROP CONSTRAINT IF EXISTS scope_required,
    ADD CONSTRAINT scope_required CHECK (
        (type IN ('docker'::deployment_type, 'opentofu'::deployment_type)) = (scope IS NULL)
    );

DROP TABLE IF EXISTS opentofu_state;

ALTER TABLE DeploymentRevision
    DROP COLUMN IF EXISTS tofu_vars,
    DROP COLUMN IF EXISTS tofu_backend_config,

ALTER TABLE ApplicationVersion
    DROP COLUMN IF EXISTS tofu_config_url,
    DROP COLUMN IF EXISTS tofu_config_version;

-- PostgreSQL does not support removing enum values
-- The 'opentofu' value will remain
