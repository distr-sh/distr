-- Every sensitive value moves from its plaintext column into a BYTEA column holding the ciphertext
-- of internal/dbcrypto. A value lives in exactly one of the two, never in both, which the
-- num_nonnulls check enforces. After this migration the application only ever writes the _enc
-- column; the plaintext columns remain readable until `distr maintenance encrypt-database` (or
-- DATABASE_ENCRYPTION_MIGRATE_ON_BOOT) has moved existing rows over, and are then always NULL.

ALTER TABLE Secret
  ADD COLUMN value_enc BYTEA,
  ALTER COLUMN value DROP NOT NULL,
  ADD CONSTRAINT Secret_value_encryption CHECK (num_nonnulls(value, value_enc) <= 1);

ALTER TABLE CustomOIDCConfiguration
  ADD COLUMN client_secret_enc BYTEA,
  ALTER COLUMN client_secret DROP NOT NULL,
  ALTER COLUMN client_secret DROP DEFAULT,
  ADD CONSTRAINT CustomOIDCConfiguration_client_secret_encryption
    CHECK (num_nonnulls(client_secret, client_secret_enc) <= 1);

ALTER TABLE CustomEmailConfiguration
  ADD COLUMN smtp_password_enc BYTEA,
  ALTER COLUMN smtp_password DROP NOT NULL,
  ALTER COLUMN smtp_password DROP DEFAULT,
  ADD CONSTRAINT CustomEmailConfiguration_smtp_password_encryption
    CHECK (num_nonnulls(smtp_password, smtp_password_enc) <= 1);

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
    CHECK (num_nonnulls(content, content_enc) <= 1);

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
  ADD COLUMN registry_password_enc BYTEA,
  ADD CONSTRAINT ApplicationEntitlement_registry_password_encryption
    CHECK (num_nonnulls(registry_password, registry_password_enc) <= 1);

-- Same widening for the constraint that keeps the registry credentials all set or all unset. The
-- old name is the one Postgres generated for the unnamed CHECK in 12_application_license.
ALTER TABLE ApplicationEntitlement DROP CONSTRAINT applicationlicense_check;
ALTER TABLE ApplicationEntitlement ADD CONSTRAINT ApplicationEntitlement_registry_credentials CHECK (
  (
    registry_url IS NULL AND registry_username IS NULL
    AND num_nonnulls(registry_password, registry_password_enc) = 0
  )
  OR (
    registry_url IS NOT NULL AND registry_username IS NOT NULL
    AND num_nonnulls(registry_password, registry_password_enc) = 1
  )
);

ALTER TABLE SupportBundle
  ADD COLUMN bundle_secret_enc BYTEA,
  ALTER COLUMN bundle_secret DROP NOT NULL,
  ADD CONSTRAINT SupportBundle_bundle_secret_encryption
    CHECK (num_nonnulls(bundle_secret, bundle_secret_enc) <= 1);

-- An access token is the only credential here that a query looks up by value alone, with no id to
-- narrow the row down first, so it cannot be encrypted with a fresh nonce per write. It is replaced
-- by a keyed hash, which keeps the lookup exact and the unique constraint intact while making the
-- stored value useless to anyone holding a database dump. Nothing ever needs the token back: it is
-- returned to the user once, when it is created. Unlike the columns above it must therefore always
-- have exactly one representation.
ALTER TABLE AccessToken
  ADD COLUMN key_hmac BYTEA,
  ALTER COLUMN key DROP NOT NULL,
  ADD CONSTRAINT AccessToken_key_encryption CHECK (num_nonnulls(key, key_hmac) = 1);

CREATE UNIQUE INDEX AccessToken_key_hmac ON AccessToken (key_hmac);

-- These two tables are the only ones here that grow without bound, so the encryption migration and
-- the startup check that reports leftover plaintext get a partial index instead of a sequential
-- scan. Both indexes are empty once every row is encrypted.
CREATE INDEX DeploymentRevision_unencrypted ON DeploymentRevision (id)
  WHERE values_yaml IS NOT NULL OR env_file_data IS NOT NULL;

CREATE INDEX SupportBundleResource_unencrypted ON SupportBundleResource (id)
  WHERE content IS NOT NULL;
