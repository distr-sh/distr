CREATE TABLE SupportBundleConfiguration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL UNIQUE REFERENCES Organization (id) ON DELETE CASCADE
);

CREATE TABLE SupportBundleConfigurationEnvVar (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    support_bundle_configuration_id UUID NOT NULL REFERENCES SupportBundleConfiguration (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    redacted BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_support_bundle_config_env_var_config_id
    ON SupportBundleConfigurationEnvVar (support_bundle_configuration_id);

CREATE TYPE support_bundle_status AS ENUM ('initialized', 'created', 'resolved');

CREATE TABLE SupportBundle (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    customer_organization_id UUID NOT NULL REFERENCES CustomerOrganization (id) ON DELETE CASCADE,
    created_by_user_account_id UUID NOT NULL REFERENCES UserAccount (id),
    title TEXT,
    description TEXT,
    status support_bundle_status NOT NULL DEFAULT 'initialized',
    access_token_id UUID REFERENCES AccessToken (id) ON DELETE SET NULL
);

CREATE INDEX idx_support_bundle_org_id ON SupportBundle (organization_id);
CREATE INDEX idx_support_bundle_customer_org_id ON SupportBundle (customer_organization_id);

CREATE TABLE SupportBundleResource (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    support_bundle_id UUID NOT NULL REFERENCES SupportBundle (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    content TEXT NOT NULL
);

CREATE INDEX idx_support_bundle_resource_bundle_id
    ON SupportBundleResource (support_bundle_id);

CREATE TABLE SupportBundleComment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    support_bundle_id UUID NOT NULL REFERENCES SupportBundle (id) ON DELETE CASCADE,
    user_account_id UUID NOT NULL REFERENCES UserAccount (id),
    content TEXT NOT NULL
);

CREATE INDEX idx_support_bundle_comment_bundle_id
    ON SupportBundleComment (support_bundle_id);
