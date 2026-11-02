ALTER TABLE SupportBundle ADD COLUMN deployment_type DEPLOYMENT_TYPE NOT NULL DEFAULT 'docker';
ALTER TABLE SupportBundleConfigurationScript ADD COLUMN deployment_type DEPLOYMENT_TYPE NOT NULL DEFAULT 'docker';

DROP INDEX idx_support_bundle_configuration_script_org_id_name;
CREATE UNIQUE INDEX idx_support_bundle_configuration_script_org_id_type_name
    ON SupportBundleConfigurationScript (organization_id, deployment_type, lower(name));
