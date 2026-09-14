-- Rolling back cannot recover any value that has already been encrypted, because SQL has no access
-- to the key. Run `distr maintenance decrypt-database` before this to move every encrypted value
-- back into its plaintext column, which makes the rollback lossless. What is left encrypted without
-- it is replaced with an empty value below so that the NOT NULL constraints can be restored, except
-- for the support bundle secret, which is given a fresh random value instead so that no two bundles
-- share one.
--
-- Every one of those updates has to run after its num_nonnulls constraint is dropped and before its
-- _enc column is: writing the plaintext column of an encrypted row is exactly the state the
-- constraint rejects, and the updates that tell an encrypted value from an absent one read the _enc
-- column to do it. Reordering either way breaks the rollback of any database that holds data.

DROP INDEX SupportBundleResource_unencrypted;
DROP INDEX DeploymentRevision_unencrypted;
DROP INDEX SupportBundleResource_content_key;
DROP INDEX DeploymentRevision_env_file_data_key;
DROP INDEX DeploymentRevision_values_yaml_key;

ALTER TABLE SupportBundle DROP CONSTRAINT SupportBundle_bundle_secret_encryption;

UPDATE SupportBundle SET bundle_secret = encode(gen_random_uuid()::TEXT::BYTEA, 'hex')
  WHERE bundle_secret IS NULL;

ALTER TABLE SupportBundle
  DROP COLUMN bundle_secret_enc,
  ALTER COLUMN bundle_secret SET NOT NULL;

ALTER TABLE ApplicationEntitlement
  DROP CONSTRAINT ApplicationEntitlement_registry_credentials,
  DROP CONSTRAINT ApplicationEntitlement_registry_username_encryption,
  DROP CONSTRAINT ApplicationEntitlement_registry_password_encryption;

UPDATE ApplicationEntitlement SET registry_username = '' WHERE registry_username_enc IS NOT NULL;
UPDATE ApplicationEntitlement SET registry_password = '' WHERE registry_password_enc IS NOT NULL;

ALTER TABLE ApplicationEntitlement
  DROP COLUMN registry_username_enc,
  DROP COLUMN registry_password_enc;

ALTER TABLE ApplicationEntitlement ADD CONSTRAINT applicationlicense_check CHECK (
  (registry_url IS NULL AND registry_username IS NULL AND registry_password IS NULL)
  OR (registry_url IS NOT NULL AND registry_username IS NOT NULL AND registry_password IS NOT NULL)
);

ALTER TABLE UserAccount
  DROP CONSTRAINT mfa_secret_not_null_if_enabled,
  DROP CONSTRAINT UserAccount_mfa_secret_encryption;

UPDATE UserAccount SET mfa_secret = '' WHERE mfa_secret_enc IS NOT NULL;

ALTER TABLE UserAccount DROP COLUMN mfa_secret_enc;

ALTER TABLE UserAccount ADD CONSTRAINT mfa_secret_not_null_if_enabled
  CHECK (mfa_enabled = false OR mfa_secret IS NOT NULL);

ALTER TABLE Organization
  DROP CONSTRAINT Organization_stripe_webhook_secret_encryption,
  DROP COLUMN stripe_webhook_secret_enc;

ALTER TABLE SupportBundleResource DROP CONSTRAINT SupportBundleResource_content_encryption;

UPDATE SupportBundleResource SET content = '' WHERE content IS NULL;

ALTER TABLE SupportBundleResource
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

ALTER TABLE CustomEmailConfiguration
  DROP CONSTRAINT CustomEmailConfiguration_smtp_username_encryption,
  DROP CONSTRAINT CustomEmailConfiguration_smtp_password_encryption;

UPDATE CustomEmailConfiguration SET smtp_username = '' WHERE smtp_username IS NULL;
UPDATE CustomEmailConfiguration SET smtp_password = '' WHERE smtp_password IS NULL;

ALTER TABLE CustomEmailConfiguration
  DROP COLUMN smtp_username_enc,
  DROP COLUMN smtp_password_enc,
  ALTER COLUMN smtp_username SET DEFAULT '',
  ALTER COLUMN smtp_username SET NOT NULL,
  ALTER COLUMN smtp_password SET DEFAULT '',
  ALTER COLUMN smtp_password SET NOT NULL;

ALTER TABLE CustomOIDCConfiguration DROP CONSTRAINT CustomOIDCConfiguration_client_secret_encryption;

UPDATE CustomOIDCConfiguration SET client_secret = '' WHERE client_secret IS NULL;

ALTER TABLE CustomOIDCConfiguration
  DROP COLUMN client_secret_enc,
  ALTER COLUMN client_secret SET DEFAULT '',
  ALTER COLUMN client_secret SET NOT NULL;

ALTER TABLE Secret DROP CONSTRAINT Secret_value_encryption;

UPDATE Secret SET value = '' WHERE value IS NULL;

ALTER TABLE Secret
  DROP COLUMN value_enc,
  ALTER COLUMN value SET NOT NULL;
