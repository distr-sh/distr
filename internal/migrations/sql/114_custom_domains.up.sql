ALTER TYPE FEATURE ADD VALUE IF NOT EXISTS 'custom_domains';

CREATE TYPE CUSTOM_DOMAIN_TYPE AS ENUM ('app', 'registry');

-- Self-service custom domains served by the Caddy deployment via on-demand TLS.
-- The legacy OrganizationBranding.app_domain / registry_domain columns stay untouched
-- for now and keep working as a fallback; migrating their values into this table is
-- handled by a later follow-on migration.
CREATE TABLE CustomDomain (
  id              UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at      TIMESTAMP          NOT NULL DEFAULT current_timestamp,
  -- bare lowercase hostname, without scheme
  domain          TEXT               NOT NULL,
  -- which endpoint the domain primarily fronts. Registry rows are optional, because an
  -- app domain serves registry traffic as well (Caddy routes /v2/ to the registry).
  domain_type     CUSTOM_DOMAIN_TYPE NOT NULL,
  organization_id UUID               NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  -- a domain may exist only once, globally; the backing index is also what the Caddy
  -- on-demand TLS "ask" lookup runs against during TLS handshakes
  CONSTRAINT CustomDomain_domain_unique UNIQUE (domain),
  CONSTRAINT CustomDomain_organization_domain_type_unique UNIQUE (organization_id, domain_type)
);

CREATE INDEX fk_CustomDomain_organization_id ON CustomDomain(organization_id);
