-- Every sensitive value moves from its plaintext column into a BYTEA column holding the ciphertext
-- of internal/dbcrypto. After this migration the application only ever writes the _enc column; the
-- plaintext columns remain readable until `distr maintenance encrypt-database` (or
-- DATABASE_ENCRYPTION_MIGRATE_ON_BOOT) has moved existing rows over, and are then always NULL.
--
-- A num_nonnulls check keeps the nullability the column had before this migration, since dropping
-- NOT NULL from a plaintext column would otherwise lose it: = 1 where the value is required, so
-- that it lives in exactly one of the two columns, and <= 1 where it is optional, so that it may
-- also be absent from both. Neither ever holds a value in both at once.

ALTER TABLE Secret
  ADD COLUMN value_enc BYTEA,
  ALTER COLUMN value DROP NOT NULL,
  ADD CONSTRAINT Secret_value_encryption CHECK (num_nonnulls(value, value_enc) = 1);

ALTER TABLE CustomOIDCConfiguration
  ADD COLUMN client_secret_enc BYTEA,
  ALTER COLUMN client_secret DROP NOT NULL,
  ALTER COLUMN client_secret DROP DEFAULT,
  ADD CONSTRAINT CustomOIDCConfiguration_client_secret_encryption
    CHECK (num_nonnulls(client_secret, client_secret_enc) = 1);

ALTER TABLE CustomEmailConfiguration
  ADD COLUMN smtp_username_enc BYTEA,
  ADD COLUMN smtp_password_enc BYTEA,
  ALTER COLUMN smtp_username DROP NOT NULL,
  ALTER COLUMN smtp_username DROP DEFAULT,
  ALTER COLUMN smtp_password DROP NOT NULL,
  ALTER COLUMN smtp_password DROP DEFAULT,
  ADD CONSTRAINT CustomEmailConfiguration_smtp_username_encryption
    CHECK (num_nonnulls(smtp_username, smtp_username_enc) = 1),
  ADD CONSTRAINT CustomEmailConfiguration_smtp_password_encryption
    CHECK (num_nonnulls(smtp_password, smtp_password_enc) = 1);

ALTER TABLE Artifact
  ADD COLUMN upstream_username_enc BYTEA,
  ADD COLUMN upstream_password_enc BYTEA,
  ADD CONSTRAINT Artifact_upstream_username_encryption
    CHECK (num_nonnulls(upstream_username, upstream_username_enc) <= 1),
  ADD CONSTRAINT Artifact_upstream_password_encryption
    CHECK (num_nonnulls(upstream_password, upstream_password_enc) <= 1);

ALTER TABLE DeploymentRevision
  ADD COLUMN values_yaml_enc BYTEA,
  ADD COLUMN env_file_data_enc BYTEA,
  ADD CONSTRAINT DeploymentRevision_values_yaml_encryption
    CHECK (num_nonnulls(values_yaml, values_yaml_enc) <= 1),
  ADD CONSTRAINT DeploymentRevision_env_file_data_encryption
    CHECK (num_nonnulls(env_file_data, env_file_data_enc) <= 1);

ALTER TABLE SupportBundleResource
  ADD COLUMN content_enc BYTEA,
  ALTER COLUMN content DROP NOT NULL,
  ADD CONSTRAINT SupportBundleResource_content_encryption
    CHECK (num_nonnulls(content, content_enc) = 1);

ALTER TABLE Organization
  ADD COLUMN stripe_webhook_secret_enc BYTEA,
  ADD CONSTRAINT Organization_stripe_webhook_secret_encryption
    CHECK (num_nonnulls(stripe_webhook_secret, stripe_webhook_secret_enc) <= 1);

ALTER TABLE UserAccount
  ADD COLUMN mfa_secret_enc BYTEA,
  ADD CONSTRAINT UserAccount_mfa_secret_encryption
    CHECK (num_nonnulls(mfa_secret, mfa_secret_enc) <= 1);

-- Widened so that an enabled second factor is satisfied by either representation of the secret.
ALTER TABLE UserAccount DROP CONSTRAINT mfa_secret_not_null_if_enabled;
ALTER TABLE UserAccount ADD CONSTRAINT mfa_secret_not_null_if_enabled
  CHECK (mfa_enabled = false OR num_nonnulls(mfa_secret, mfa_secret_enc) = 1);

ALTER TABLE ApplicationEntitlement
  ADD COLUMN registry_username_enc BYTEA,
  ADD COLUMN registry_password_enc BYTEA,
  ADD CONSTRAINT ApplicationEntitlement_registry_username_encryption
    CHECK (num_nonnulls(registry_username, registry_username_enc) <= 1),
  ADD CONSTRAINT ApplicationEntitlement_registry_password_encryption
    CHECK (num_nonnulls(registry_password, registry_password_enc) <= 1);

-- Same widening for the constraint that keeps the registry credentials all set or all unset. The
-- old name is the one Postgres generated for the unnamed CHECK in 12_application_license.
ALTER TABLE ApplicationEntitlement DROP CONSTRAINT applicationlicense_check;
ALTER TABLE ApplicationEntitlement ADD CONSTRAINT ApplicationEntitlement_registry_credentials CHECK (
  (
    registry_url IS NULL
    AND num_nonnulls(registry_username, registry_username_enc) = 0
    AND num_nonnulls(registry_password, registry_password_enc) = 0
  )
  OR (
    registry_url IS NOT NULL
    AND num_nonnulls(registry_username, registry_username_enc) = 1
    AND num_nonnulls(registry_password, registry_password_enc) = 1
  )
);

ALTER TABLE SupportBundle
  ADD COLUMN bundle_secret_enc BYTEA,
  ALTER COLUMN bundle_secret DROP NOT NULL,
  ADD CONSTRAINT SupportBundle_bundle_secret_encryption
    CHECK (num_nonnulls(bundle_secret, bundle_secret_enc) = 1);

-- These two tables are the only ones here that grow without bound, so the encryption migration and
-- the startup check that reports leftover plaintext get a partial index instead of a sequential
-- scan. Both indexes are empty once every row is encrypted.
CREATE INDEX DeploymentRevision_unencrypted ON DeploymentRevision (id)
  WHERE values_yaml IS NOT NULL OR env_file_data IS NOT NULL;

CREATE INDEX SupportBundleResource_unencrypted ON SupportBundleResource (id)
  WHERE content IS NOT NULL;

-- The first two bytes of a stored value are its format version and key id. Indexing them turns the
-- startup check for a retired key into two index lookups, instead of a scan that fetches every
-- value out of the TOAST table. The expression has to stay identical to the one in internal/db.
CREATE INDEX DeploymentRevision_values_yaml_key
  ON DeploymentRevision (substring(values_yaml_enc FROM 1 FOR 2)) WHERE values_yaml_enc IS NOT NULL;

CREATE INDEX DeploymentRevision_env_file_data_key
  ON DeploymentRevision (substring(env_file_data_enc FROM 1 FOR 2)) WHERE env_file_data_enc IS NOT NULL;

CREATE INDEX SupportBundleResource_content_key
  ON SupportBundleResource (substring(content_enc FROM 1 FOR 2)) WHERE content_enc IS NOT NULL;
