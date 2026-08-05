ALTER TABLE OIDCState
  DROP COLUMN custom_oidc_configuration_id,
  DROP COLUMN nonce;

DROP INDEX UserAccountOIDCIdentity_config_issuer_subject_uq;
CREATE UNIQUE INDEX UserAccountOIDCIdentity_issuer_subject_uq
  ON UserAccountOIDCIdentity (issuer, subject);

DROP INDEX fk_UserAccountOIDCIdentity_custom_oidc_configuration_id;

ALTER TABLE UserAccountOIDCIdentity
  DROP CONSTRAINT UserAccountOIDCIdentity_custom_config_check,
  DROP COLUMN custom_oidc_configuration_id;

DROP TABLE CustomOIDCConfiguration;
