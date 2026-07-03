ALTER TABLE OrganizationBranding
  ADD COLUMN app_domain TEXT DEFAULT NULL,
  ADD COLUMN registry_domain TEXT DEFAULT NULL,
  ADD COLUMN email_from_address TEXT DEFAULT NULL,
  ADD COLUMN logo_image_id UUID REFERENCES File (id) ON DELETE SET NULL;

INSERT INTO OrganizationBranding (organization_id, app_domain, registry_domain, email_from_address)
SELECT o.id, o.app_domain, o.registry_domain, o.email_from_address
FROM Organization o
WHERE (o.app_domain IS NOT NULL OR o.registry_domain IS NOT NULL OR o.email_from_address IS NOT NULL)
  AND NOT EXISTS (SELECT 1 FROM OrganizationBranding b WHERE b.organization_id = o.id);

UPDATE OrganizationBranding b
SET app_domain = o.app_domain,
    registry_domain = o.registry_domain,
    email_from_address = o.email_from_address
FROM Organization o
WHERE b.organization_id = o.id
  AND (o.app_domain IS NOT NULL OR o.registry_domain IS NOT NULL OR o.email_from_address IS NOT NULL);

WITH ins AS (
  INSERT INTO File (organization_id, content_type, data, file_name, file_size)
  SELECT organization_id,
         COALESCE(logo_content_type, 'application/octet-stream'),
         logo,
         COALESCE(logo_file_name, 'logo'),
         octet_length(logo)
  FROM OrganizationBranding
  WHERE logo IS NOT NULL
  RETURNING id, organization_id
)
UPDATE OrganizationBranding b
SET logo_image_id = ins.id
FROM ins
WHERE ins.organization_id = b.organization_id;

ALTER TABLE Organization
  DROP COLUMN app_domain,
  DROP COLUMN registry_domain,
  DROP COLUMN email_from_address;

ALTER TABLE OrganizationBranding
  DROP COLUMN logo,
  DROP COLUMN logo_file_name,
  DROP COLUMN logo_content_type;
