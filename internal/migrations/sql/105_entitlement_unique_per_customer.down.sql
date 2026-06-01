ALTER TABLE LicenseKey
  DROP CONSTRAINT licensekey_name_unique;
ALTER TABLE LicenseKey
  ADD CONSTRAINT licensekey_organization_id_name_key
    UNIQUE (organization_id, name);

ALTER TABLE ArtifactEntitlement
  DROP CONSTRAINT artifactentitlement_name_unique;
ALTER TABLE ArtifactEntitlement
  ADD CONSTRAINT artifactentitlement_name_unique
    UNIQUE (organization_id, name);

ALTER TABLE ApplicationEntitlement
  DROP CONSTRAINT applicationentitlement_name_unique;
ALTER TABLE ApplicationEntitlement
  ADD CONSTRAINT applicationlicense_name_organization_id_key
    UNIQUE (name, organization_id);
