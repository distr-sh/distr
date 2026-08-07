-- CustomDomain is referenced with ON DELETE RESTRICT, so the configurations of the domains that go
-- away have to be deleted before the domains themselves
DELETE FROM CustomOIDCConfiguration c
  USING CustomDomain d
  WHERE c.custom_domain_id = d.id
    AND (d.customer_organization_id IS NOT NULL OR d.domain_type = 'customer_portal');

DELETE FROM CustomDomain WHERE customer_organization_id IS NOT NULL OR domain_type = 'customer_portal';

DROP INDEX CustomOIDCConfiguration_sp_initiated_uq;
CREATE UNIQUE INDEX CustomOIDCConfiguration_sp_initiated_uq
  ON CustomOIDCConfiguration (organization_id) WHERE sp_initiated;

DROP INDEX CustomDomain_customer_organization_uq;
DROP INDEX CustomDomain_organization_domain_type_uq;

ALTER TABLE CustomDomain
  ADD CONSTRAINT CustomDomain_organization_domain_type_unique UNIQUE (organization_id, domain_type);

DROP INDEX fk_CustomDomain_customer_organization_id;

ALTER TABLE CustomDomain
  DROP CONSTRAINT CustomDomain_customer_organization_type_check,
  DROP COLUMN customer_organization_id;
