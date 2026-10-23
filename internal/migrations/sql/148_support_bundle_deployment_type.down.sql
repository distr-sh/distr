DROP INDEX idx_support_bundle_configuration_script_org_id_type_name;
DELETE FROM SupportBundleConfigurationScript WHERE deployment_type <> 'docker';
CREATE UNIQUE INDEX idx_support_bundle_configuration_script_org_id_name
    ON SupportBundleConfigurationScript (organization_id, lower(name));
ALTER TABLE SupportBundleConfigurationScript DROP COLUMN deployment_type;

ALTER TABLE SupportBundle DROP COLUMN deployment_type;
