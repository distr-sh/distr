DROP INDEX IF EXISTS idx_OrganizationBranding_app_domain;

ALTER TABLE OrganizationBranding
  DROP COLUMN favicon_image_id,
  DROP COLUMN page_title;

ALTER TABLE File
  DROP COLUMN public;
