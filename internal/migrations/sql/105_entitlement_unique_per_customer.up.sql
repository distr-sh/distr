-- Make entitlement and license key names unique per customer organization
-- instead of per (vendor) organization. The customer organization is nullable
-- for vendor-level rows, so NULLS NOT DISTINCT keeps those names unique within
-- the vendor organization while still allowing the same name across different
-- customer organizations.

ALTER TABLE ApplicationEntitlement
  DROP CONSTRAINT applicationlicense_name_organization_id_key;
ALTER TABLE ApplicationEntitlement
  ADD CONSTRAINT applicationentitlement_name_unique
    UNIQUE NULLS NOT DISTINCT (organization_id, customer_organization_id, name);

ALTER TABLE ArtifactEntitlement
  DROP CONSTRAINT artifactentitlement_name_unique;
ALTER TABLE ArtifactEntitlement
  ADD CONSTRAINT artifactentitlement_name_unique
    UNIQUE NULLS NOT DISTINCT (organization_id, customer_organization_id, name);

ALTER TABLE LicenseKey
  DROP CONSTRAINT licensekey_organization_id_name_key;
ALTER TABLE LicenseKey
  ADD CONSTRAINT licensekey_name_unique
    UNIQUE NULLS NOT DISTINCT (organization_id, customer_organization_id, name);
