-- A customer_portal domain with a NULL customer_organization_id is the vendor's shared portal for all
-- of its customers; with it set, it is one customer's own domain. app and registry domains are always
-- vendor-scoped, which is what keeps a customer hostname out of agent manifests and the registry host.
ALTER TABLE CustomDomain
  ADD COLUMN customer_organization_id UUID REFERENCES CustomerOrganization (id) ON DELETE CASCADE,
  ADD CONSTRAINT CustomDomain_customer_organization_type_check
    CHECK (customer_organization_id IS NULL OR domain_type = 'customer_portal');

CREATE INDEX fk_CustomDomain_customer_organization_id ON CustomDomain (customer_organization_id);

ALTER TABLE CustomDomain DROP CONSTRAINT CustomDomain_organization_domain_type_unique;

CREATE UNIQUE INDEX CustomDomain_organization_domain_type_uq
  ON CustomDomain (organization_id, domain_type) WHERE customer_organization_id IS NULL;

CREATE UNIQUE INDEX CustomDomain_customer_organization_uq
  ON CustomDomain (organization_id, customer_organization_id) WHERE customer_organization_id IS NOT NULL;

-- automatic sign-in is a property of a host, and an organization now has several
DROP INDEX CustomOIDCConfiguration_sp_initiated_uq;
CREATE UNIQUE INDEX CustomOIDCConfiguration_sp_initiated_uq
  ON CustomOIDCConfiguration (custom_domain_id) WHERE sp_initiated;
