CREATE TABLE LicenseTemplate (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
    name TEXT NOT NULL,
    organization_id UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
    payload_template TEXT NOT NULL,
    expiration_grace_period_days INTEGER NOT NULL DEFAULT 0
      CHECK (expiration_grace_period_days >= 0),
    UNIQUE (organization_id, name)
);

CREATE INDEX ON LicenseTemplate(organization_id);

ALTER TABLE Organization ADD COLUMN stripe_webhook_secret TEXT;

ALTER TABLE LicenseKey ADD COLUMN license_template_id UUID REFERENCES LicenseTemplate(id) ON DELETE SET NULL;

ALTER TYPE FEATURE ADD VALUE IF NOT EXISTS 'vendor_billing';
