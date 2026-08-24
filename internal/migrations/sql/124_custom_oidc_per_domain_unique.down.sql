-- Deliberately no cleanup of names or slugs used on more than one domain: adding the constraints then
-- fails and the rollback stops, which is what we want. Never resolve the conflict by deleting a
-- provider, that would silently take a working login away from whoever configured it.
ALTER TABLE CustomOIDCConfiguration
  DROP CONSTRAINT CustomOIDCConfiguration_domain_name_unique,
  DROP CONSTRAINT CustomOIDCConfiguration_domain_slug_unique,
  ADD CONSTRAINT CustomOIDCConfiguration_org_name_unique UNIQUE (organization_id, name),
  ADD CONSTRAINT CustomOIDCConfiguration_org_slug_unique UNIQUE (organization_id, slug);
