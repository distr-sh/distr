ALTER TABLE File
  ADD COLUMN public BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE OrganizationBranding
  ADD COLUMN page_title       TEXT DEFAULT NULL,
  ADD COLUMN favicon_image_id UUID DEFAULT NULL REFERENCES File (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_OrganizationBranding_app_domain
  ON OrganizationBranding (app_domain)
  WHERE app_domain IS NOT NULL;
