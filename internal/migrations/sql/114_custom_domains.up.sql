ALTER TYPE FEATURE ADD VALUE IF NOT EXISTS 'custom_domains';

-- Self-service custom domains served via the Caddy deployment (on-demand TLS).
-- The legacy OrganizationBranding.app_domain / registry_domain columns stay untouched
-- for now and keep working as a fallback; migrating their values into this table is
-- handled by a later follow-on migration.
CREATE TABLE CustomDomain (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  -- bare lowercase hostname, no scheme
  domain TEXT NOT NULL,
  -- which endpoint this domain primarily fronts. Registry rows are optional:
  -- an app domain serves registry traffic too (/v2/ path routing in Caddy).
  domain_type TEXT NOT NULL CHECK (domain_type IN ('app', 'registry')),
  -- the vendor organization always owns and administers the domain
  organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
  -- optional narrower scope: when set, the domain is dedicated to one customer or
  -- partner organization. When both are NULL, the domain is the org-wide domain
  -- shared by the vendor and all of its customers and partners.
  customer_organization_id UUID REFERENCES CustomerOrganization (id) ON DELETE CASCADE,
  partner_organization_id UUID REFERENCES PartnerOrganization (id) ON DELETE CASCADE,
  CONSTRAINT CustomDomain_at_most_one_scope
    CHECK (num_nonnulls(customer_organization_id, partner_organization_id) <= 1),
  -- a domain may exist only once, globally; the backing index is also what the
  -- Caddy on-demand TLS "ask" lookup runs against during TLS handshakes
  CONSTRAINT CustomDomain_domain_unique UNIQUE (domain)
);

CREATE INDEX fk_CustomDomain_organization_id ON CustomDomain (organization_id);

-- at most one org-wide (unscoped) domain per org and type,
-- and at most one domain per customer/partner scope and type
CREATE UNIQUE INDEX CustomDomain_organization_type
  ON CustomDomain (organization_id, domain_type)
  WHERE customer_organization_id IS NULL AND partner_organization_id IS NULL;
CREATE UNIQUE INDEX CustomDomain_customer_organization_type
  ON CustomDomain (customer_organization_id, domain_type)
  WHERE customer_organization_id IS NOT NULL;
CREATE UNIQUE INDEX CustomDomain_partner_organization_type
  ON CustomDomain (partner_organization_id, domain_type)
  WHERE partner_organization_id IS NOT NULL;
