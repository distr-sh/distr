ALTER TYPE FEATURE ADD VALUE IF NOT EXISTS 'custom_emails';

-- The legacy OrganizationBranding.email_from_address column stays untouched and keeps working as a
-- fallback for the from address, just like the legacy branding domain columns do for CustomDomain.
CREATE TABLE CustomEmailConfiguration (
  id                         UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at                 TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_at                 TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_by_user_account_id UUID      REFERENCES UserAccount (id) ON DELETE SET NULL,
  organization_id            UUID      NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
  -- a stored but disabled configuration falls back to the instance mailer, so that a plan
  -- downgrade or a broken provider can be handled without losing the configuration
  enabled                    BOOLEAN   NOT NULL DEFAULT TRUE,
  from_address               TEXT      NOT NULL,
  smtp_host                  TEXT      NOT NULL,
  smtp_port                  INT       NOT NULL,
  smtp_username              TEXT      NOT NULL DEFAULT '',
  smtp_password              TEXT      NOT NULL DEFAULT '',
  smtp_implicit_tls          BOOLEAN   NOT NULL DEFAULT FALSE,
  -- exactly one configuration per organization; the backing index also serves the per-org lookup
  CONSTRAINT CustomEmailConfiguration_organization_unique UNIQUE (organization_id)
);
