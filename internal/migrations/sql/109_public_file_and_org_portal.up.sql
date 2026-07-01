ALTER TABLE File
  ADD COLUMN public BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE Organization
  ADD COLUMN page_title      TEXT DEFAULT NULL,
  ADD COLUMN favicon_image_id UUID DEFAULT NULL REFERENCES File (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS fk_Organization_favicon_image_id ON Organization (favicon_image_id);

CREATE INDEX IF NOT EXISTS idx_Organization_app_domain_normalized
  ON Organization (lower(regexp_replace(app_domain, '^https?://', '')));
