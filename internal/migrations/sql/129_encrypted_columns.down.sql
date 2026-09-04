-- Rolling back cannot recover any value that has already been encrypted: SQL has no access to the
-- key, and the access token and support bundle secrets are keyed hashes that are one-way by design.
-- Decrypt first if the data matters. What is left encrypted is replaced with an empty value below so
-- that the NOT NULL constraints can be restored, and migrated access tokens are deleted rather than
-- given a fabricated key.

DROP INDEX SupportBundleResource_unencrypted;
DROP INDEX DeploymentRevision_unencrypted;

DELETE FROM AccessToken WHERE key IS NULL;

DROP INDEX AccessToken_key_hmac;

ALTER TABLE AccessToken
  DROP CONSTRAINT AccessToken_key_encryption,
  DROP COLUMN key_hmac,
  ALTER COLUMN key SET NOT NULL;

UPDATE SupportBundle SET bundle_secret = encode(gen_random_uuid()::TEXT::BYTEA, 'hex')
  WHERE bundle_secret IS NULL;

ALTER TABLE SupportBundle
  DROP CONSTRAINT SupportBundle_bundle_secret_encryption,
  DROP COLUMN bundle_secret_enc,
  ALTER COLUMN bundle_secret SET NOT NULL;

ALTER TABLE ApplicationEntitlement DROP CONSTRAINT ApplicationEntitlement_registry_credentials;
UPDATE ApplicationEntitlement SET registry_password = '' WHERE registry_password_enc IS NOT NULL;
ALTER TABLE ApplicationEntitlement ADD CONSTRAINT applicationlicense_check CHECK (
  (registry_url IS NULL AND registry_username IS NULL AND registry_password IS NULL)
  OR (registry_url IS NOT NULL AND registry_username IS NOT NULL AND registry_password IS NOT NULL)
);

ALTER TABLE ApplicationEntitlement
  DROP CONSTRAINT ApplicationEntitlement_registry_password_encryption,
  DROP COLUMN registry_password_enc;

ALTER TABLE UserAccount DROP CONSTRAINT mfa_secret_not_null_if_enabled;
UPDATE UserAccount SET mfa_secret = '' WHERE mfa_secret_enc IS NOT NULL;
ALTER TABLE UserAccount ADD CONSTRAINT mfa_secret_not_null_if_enabled
  CHECK (mfa_enabled = false OR mfa_secret IS NOT NULL);

ALTER TABLE UserAccount
  DROP CONSTRAINT UserAccount_mfa_secret_encryption,
  DROP COLUMN mfa_secret_enc;

ALTER TABLE Organization
  DROP CONSTRAINT Organization_stripe_webhook_secret_encryption,
  DROP COLUMN stripe_webhook_secret_enc;

UPDATE SupportBundleResource SET content = '' WHERE content IS NULL;

ALTER TABLE SupportBundleResource
  DROP CONSTRAINT SupportBundleResource_content_encryption,
  DROP COLUMN content_enc,
  ALTER COLUMN content SET NOT NULL;

ALTER TABLE DeploymentRevision
  DROP CONSTRAINT DeploymentRevision_env_file_data_encryption,
  DROP CONSTRAINT DeploymentRevision_values_yaml_encryption,
  DROP COLUMN env_file_data_enc,
  DROP COLUMN values_yaml_enc;

ALTER TABLE Artifact
  DROP CONSTRAINT Artifact_upstream_password_encryption,
  DROP CONSTRAINT Artifact_upstream_username_encryption,
  DROP COLUMN upstream_password_enc,
  DROP COLUMN upstream_username_enc;

UPDATE CustomEmailConfiguration SET smtp_password = '' WHERE smtp_password IS NULL;

ALTER TABLE CustomEmailConfiguration
  DROP CONSTRAINT CustomEmailConfiguration_smtp_password_encryption,
  DROP COLUMN smtp_password_enc,
  ALTER COLUMN smtp_password SET DEFAULT '',
  ALTER COLUMN smtp_password SET NOT NULL;

UPDATE CustomOIDCConfiguration SET client_secret = '' WHERE client_secret IS NULL;

ALTER TABLE CustomOIDCConfiguration
  DROP CONSTRAINT CustomOIDCConfiguration_client_secret_encryption,
  DROP COLUMN client_secret_enc,
  ALTER COLUMN client_secret SET DEFAULT '',
  ALTER COLUMN client_secret SET NOT NULL;

UPDATE Secret SET value = '' WHERE value IS NULL;

ALTER TABLE Secret
  DROP CONSTRAINT Secret_value_encryption,
  DROP COLUMN value_enc,
  ALTER COLUMN value SET NOT NULL;
