CREATE UNIQUE INDEX idx_support_bundle_config_env_var_unique_name
    ON SupportBundleConfigurationEnvVar (support_bundle_configuration_id, lower(name));
