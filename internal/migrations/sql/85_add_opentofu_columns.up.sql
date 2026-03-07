ALTER TABLE ApplicationVersion
    ADD COLUMN IF NOT EXISTS tofu_config_url TEXT,
    ADD COLUMN IF NOT EXISTS tofu_config_version TEXT;

ALTER TABLE DeploymentRevision
    ADD COLUMN IF NOT EXISTS tofu_vars JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS tofu_backend_config JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS tofu_version TEXT;

CREATE TABLE IF NOT EXISTS opentofu_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL UNIQUE REFERENCES Deployment(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
    s3_key TEXT NOT NULL,
    lock_id TEXT,
    lock_info TEXT,
    locked_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_opentofu_state_organization_id ON opentofu_state(organization_id);

-- Update check constraints to allow NULL namespace and scope for opentofu type
ALTER TABLE DeploymentTarget
    DROP CONSTRAINT IF EXISTS namespace_required,
    ADD CONSTRAINT namespace_required CHECK (
        (type IN ('docker', 'opentofu')) = (namespace IS NULL)
    );

ALTER TABLE DeploymentTarget
    DROP CONSTRAINT IF EXISTS scope_required,
    ADD CONSTRAINT scope_required CHECK (
        (type IN ('docker', 'opentofu')) = (scope IS NULL)
    );
