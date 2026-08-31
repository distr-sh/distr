CREATE TABLE SupportBundleConfigurationScript (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX idx_support_bundle_configuration_script_org_id_name
    ON SupportBundleConfigurationScript (organization_id, lower(name));
