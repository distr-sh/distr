-- per domain and not per organization, so the same identity provider can be offered on the vendor
-- portal and the customer portal under one name and one slug
ALTER TABLE CustomOIDCConfiguration
  DROP CONSTRAINT CustomOIDCConfiguration_org_name_unique,
  DROP CONSTRAINT CustomOIDCConfiguration_org_slug_unique,
  ADD CONSTRAINT CustomOIDCConfiguration_domain_name_unique UNIQUE (custom_domain_id, name),
  ADD CONSTRAINT CustomOIDCConfiguration_domain_slug_unique UNIQUE (custom_domain_id, slug);
