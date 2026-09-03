-- Instance-wide providers configured via environment variables. Organization-scoped
-- providers add a 'custom' value plus a custom_oidc_configuration_id FK;
-- 'generic' therefore stays reserved for the single env-configured generic provider
-- behind /api/v1/auth/oidc/generic.
CREATE TYPE OIDC_PROVIDER AS ENUM ('github', 'google', 'microsoft', 'generic');

-- Identities provided by an IdP, linked to the user account they belong to. Matching a
-- login on (issuer, subject) instead of the email address keeps the link intact when the
-- email changes on either side.
CREATE TABLE UserAccountOIDCIdentity (
  id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at      TIMESTAMP     NOT NULL DEFAULT current_timestamp,
  user_account_id UUID          NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  provider        OIDC_PROVIDER NOT NULL,
  issuer          TEXT          NOT NULL,
  subject         TEXT          NOT NULL,
  -- email as reported by the IdP, for display only
  email           TEXT,
  last_login_at   TIMESTAMP
);

CREATE INDEX fk_UserAccountOIDCIdentity_user_account_id ON UserAccountOIDCIdentity(user_account_id);

-- A unique index rather than a table constraint: once organization-scoped providers
-- exist this is replaced by a config-scoped index, because an organization config may
-- point at an issuer that is also reachable via an instance provider.
CREATE UNIQUE INDEX UserAccountOIDCIdentity_issuer_subject_uq
  ON UserAccountOIDCIdentity (issuer, subject);
