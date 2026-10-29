DROP INDEX idx_support_bundle_configuration_script_org_id_type_name;
DELETE FROM SupportBundleConfigurationScript WHERE deployment_type <> 'docker';
CREATE UNIQUE INDEX idx_support_bundle_configuration_script_org_id_name
    ON SupportBundleConfigurationScript (organization_id, lower(name));
ALTER TABLE SupportBundleConfigurationScript DROP COLUMN deployment_type;

UPDATE SupportBundle
SET status = 'canceled', status_changed_at = now(), status_changed_by_user_account_id = NULL,
    bundle_secret_expires_at = NULL
WHERE deployment_type <> 'docker' AND status = 'initialized';
ALTER TABLE SupportBundle DROP COLUMN deployment_type;
