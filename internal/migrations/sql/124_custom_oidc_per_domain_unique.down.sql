-- the organization-wide constraints cannot be restored while a name or slug is used on more than one
-- domain, so the newer of every conflicting pair goes away, together with the identities linked to it
DELETE FROM CustomOIDCConfiguration c
  WHERE EXISTS (
    SELECT 1 FROM CustomOIDCConfiguration other
    WHERE other.organization_id = c.organization_id
      AND (other.name = c.name OR other.slug = c.slug)
      AND (other.created_at, other.id) < (c.created_at, c.id)
  );

ALTER TABLE CustomOIDCConfiguration
  DROP CONSTRAINT CustomOIDCConfiguration_domain_name_unique,
  DROP CONSTRAINT CustomOIDCConfiguration_domain_slug_unique,
  ADD CONSTRAINT CustomOIDCConfiguration_org_name_unique UNIQUE (organization_id, name),
  ADD CONSTRAINT CustomOIDCConfiguration_org_slug_unique UNIQUE (organization_id, slug);
