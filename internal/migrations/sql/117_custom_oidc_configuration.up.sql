-- OIDC providers configured by an organization for its own users, offered only on the
-- organization's custom domain.
CREATE TABLE CustomOIDCConfiguration (
  id                         UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at                 TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_at                 TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_by_user_account_id UUID      REFERENCES UserAccount (id) ON DELETE SET NULL,
  organization_id            UUID      NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
  -- the provider is only offered on this domain; ON DELETE RESTRICT so removing a domain
  -- cannot silently delete an SSO configuration (and with it every linked identity)
  custom_domain_id           UUID      NOT NULL REFERENCES CustomDomain (id) ON DELETE RESTRICT,
  name                       TEXT      NOT NULL,
  enabled                    BOOLEAN   NOT NULL DEFAULT TRUE,
  -- canonical issuer, taken from the discovery document rather than from user input
  issuer                     TEXT      NOT NULL,
  client_id                  TEXT      NOT NULL,
  client_secret              TEXT      NOT NULL DEFAULT '',
  scopes                     TEXT[]    NOT NULL DEFAULT '{openid,profile,email}',
  -- NULL means derive from the discovery document
  pkce_enabled               BOOLEAN,
  sp_initiated               BOOLEAN   NOT NULL DEFAULT FALSE,
  create_unknown_users       BOOLEAN   NOT NULL DEFAULT FALSE,
  default_user_role          USER_ROLE NOT NULL DEFAULT 'read_write',
  allowed_email_domains      TEXT[]    NOT NULL DEFAULT '{}',
  CONSTRAINT CustomOIDCConfiguration_org_name_unique UNIQUE (organization_id, name),
  -- Creating accounts for everyone the provider authenticates is only ever meant for a directory
  -- the organization owns, so it requires the domains it owns to be named.
  CONSTRAINT CustomOIDCConfiguration_provisioning_domains_check
    CHECK (NOT create_unknown_users OR cardinality(allowed_email_domains) > 0)
);

CREATE INDEX fk_CustomOIDCConfiguration_organization_id ON CustomOIDCConfiguration (organization_id);
CREATE INDEX fk_CustomOIDCConfiguration_custom_domain_id ON CustomOIDCConfiguration (custom_domain_id);

-- Only one configuration per organization may take over the login page automatically.
CREATE UNIQUE INDEX CustomOIDCConfiguration_sp_initiated_uq
  ON CustomOIDCConfiguration (organization_id) WHERE sp_initiated;

ALTER TABLE UserAccountOIDCIdentity
  ADD COLUMN custom_oidc_configuration_id UUID
    REFERENCES CustomOIDCConfiguration (id) ON DELETE CASCADE,
  ADD CONSTRAINT UserAccountOIDCIdentity_custom_config_check
    CHECK ((provider = 'custom') = (custom_oidc_configuration_id IS NOT NULL));

CREATE INDEX fk_UserAccountOIDCIdentity_custom_oidc_configuration_id
  ON UserAccountOIDCIdentity (custom_oidc_configuration_id);

-- An organization configuration may point at an issuer that is also reachable through an
-- instance provider, so identities are unique per configuration. NULLS NOT DISTINCT keeps
-- (issuer, subject) unique among the instance providers, whose configuration id is NULL.
DROP INDEX UserAccountOIDCIdentity_issuer_subject_uq;
CREATE UNIQUE INDEX UserAccountOIDCIdentity_config_issuer_subject_uq
  ON UserAccountOIDCIdentity (custom_oidc_configuration_id, issuer, subject) NULLS NOT DISTINCT;

-- The state binds a callback to the configuration that started the flow (NULL for the instance
-- providers), so a code cannot be redeemed against a different provider, and carries the nonce
-- that the ID token is checked against.
ALTER TABLE OIDCState
  ADD COLUMN custom_oidc_configuration_id UUID
    REFERENCES CustomOIDCConfiguration (id) ON DELETE CASCADE,
  ADD COLUMN nonce TEXT;
