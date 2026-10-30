CREATE TABLE SupportBundleKubernetesConfiguration (
    organization_id UUID PRIMARY KEY REFERENCES Organization (id) ON DELETE CASCADE,
    include_configmap_data BOOLEAN NOT NULL,
    include_secret_data BOOLEAN NOT NULL
);
