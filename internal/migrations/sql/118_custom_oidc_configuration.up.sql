CREATE TABLE CustomOIDCConfiguration (
  id                         UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at                 TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_at                 TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_by_user_account_id UUID      REFERENCES UserAccount (id) ON DELETE SET NULL,
  organization_id            UUID      NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
  custom_domain_id           UUID      NOT NULL REFERENCES CustomDomain (id) ON DELETE RESTRICT,
  name                       TEXT      NOT NULL,
  slug                       TEXT      NOT NULL,
  enabled                    BOOLEAN   NOT NULL DEFAULT TRUE,
  issuer                     TEXT      NOT NULL,
  client_id                  TEXT      NOT NULL,
  client_secret              TEXT      NOT NULL DEFAULT '',
  scopes                     TEXT[]    NOT NULL DEFAULT '{openid,profile,email}',
  pkce_enabled               BOOLEAN,
  sp_initiated               BOOLEAN   NOT NULL DEFAULT FALSE,
  create_unknown_users       BOOLEAN   NOT NULL DEFAULT FALSE,
  default_user_role          USER_ROLE NOT NULL DEFAULT 'read_write',
  allowed_email_domains      TEXT[]    NOT NULL DEFAULT '{}',
  CONSTRAINT CustomOIDCConfiguration_org_name_unique UNIQUE (organization_id, name),
  -- the slug is the path segment identifying the provider in its login and callback URL
  CONSTRAINT CustomOIDCConfiguration_org_slug_unique UNIQUE (organization_id, slug),
  CONSTRAINT CustomOIDCConfiguration_provisioning_domains_check
    CHECK (NOT create_unknown_users OR cardinality(allowed_email_domains) > 0)
);

CREATE INDEX fk_CustomOIDCConfiguration_organization_id ON CustomOIDCConfiguration (organization_id);
CREATE INDEX fk_CustomOIDCConfiguration_custom_domain_id ON CustomOIDCConfiguration (custom_domain_id);

CREATE UNIQUE INDEX CustomOIDCConfiguration_sp_initiated_uq
  ON CustomOIDCConfiguration (organization_id) WHERE sp_initiated;

ALTER TABLE UserAccountOIDCIdentity
  ADD COLUMN custom_oidc_configuration_id UUID
    REFERENCES CustomOIDCConfiguration (id) ON DELETE CASCADE,
  ADD CONSTRAINT UserAccountOIDCIdentity_custom_config_check
    CHECK ((provider = 'custom') = (custom_oidc_configuration_id IS NOT NULL));

CREATE INDEX fk_UserAccountOIDCIdentity_custom_oidc_configuration_id
  ON UserAccountOIDCIdentity (custom_oidc_configuration_id);

DROP INDEX UserAccountOIDCIdentity_issuer_subject_uq;
CREATE UNIQUE INDEX UserAccountOIDCIdentity_config_issuer_subject_uq
  ON UserAccountOIDCIdentity (custom_oidc_configuration_id, issuer, subject) NULLS NOT DISTINCT;

ALTER TABLE OIDCState
  ADD COLUMN custom_oidc_configuration_id UUID
    REFERENCES CustomOIDCConfiguration (id) ON DELETE CASCADE,
  ADD COLUMN nonce TEXT;
