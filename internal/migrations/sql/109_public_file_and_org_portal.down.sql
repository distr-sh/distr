DROP INDEX IF EXISTS idx_Organization_app_domain_normalized;

ALTER TABLE Organization
  DROP COLUMN favicon_image_id,
  DROP COLUMN page_title;

ALTER TABLE File
  DROP COLUMN public;
