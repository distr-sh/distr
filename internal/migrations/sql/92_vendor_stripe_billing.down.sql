ALTER TABLE LicenseKey DROP COLUMN license_template_id;
ALTER TABLE Organization DROP COLUMN stripe_webhook_secret;
DROP TABLE LicenseTemplate;
