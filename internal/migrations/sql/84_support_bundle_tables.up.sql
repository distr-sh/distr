ALTER TYPE CUSTOMER_ORGANIZATION_FEATURE ADD VALUE IF NOT EXISTS 'support_bundles';

CREATE TABLE SupportBundleConfigurationEnvVar (
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    redacted BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (organization_id, name)
);


CREATE TYPE support_bundle_status AS ENUM ('initialized', 'created', 'resolved');

CREATE TABLE SupportBundle (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    customer_organization_id UUID NOT NULL REFERENCES CustomerOrganization (id) ON DELETE CASCADE,
    created_by_user_account_id UUID NOT NULL REFERENCES UserAccount (id),
    title TEXT NOT NULL,
    description TEXT,
    status support_bundle_status NOT NULL DEFAULT 'initialized',
    collect_token_hash BYTEA,
    collect_token_expires_at TIMESTAMPTZ,
    collect_command TEXT,
    resolved_by_user_account_id UUID REFERENCES UserAccount (id)
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
