ALTER TABLE OrganizationBranding
  ADD COLUMN logo BYTEA,
  ADD COLUMN logo_file_name TEXT,
  ADD COLUMN logo_content_type TEXT;

UPDATE OrganizationBranding b
SET logo = f.data,
    logo_file_name = f.file_name,
    logo_content_type = f.content_type
FROM File f
WHERE b.logo_image_id = f.id;

ALTER TABLE Organization
  ADD COLUMN app_domain TEXT DEFAULT NULL,
  ADD COLUMN registry_domain TEXT DEFAULT NULL,
  ADD COLUMN email_from_address TEXT DEFAULT NULL;

UPDATE Organization o
SET app_domain = b.app_domain,
    registry_domain = b.registry_domain,
    email_from_address = b.email_from_address
FROM OrganizationBranding b
WHERE b.organization_id = o.id
  AND (b.app_domain IS NOT NULL OR b.registry_domain IS NOT NULL OR b.email_from_address IS NOT NULL);

ALTER TABLE OrganizationBranding
  DROP COLUMN app_domain,
  DROP COLUMN registry_domain,
  DROP COLUMN email_from_address,
  DROP COLUMN logo_image_id;
