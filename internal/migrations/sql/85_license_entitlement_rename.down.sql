-- Revert LicenseKey indexes
DROP INDEX IF EXISTS idx_licensekey_customer_organization_id;
CREATE INDEX idx_usagelicense_customer_organization_id ON LicenseKey (customer_organization_id);

DROP INDEX IF EXISTS idx_licensekey_organization_id;
CREATE INDEX idx_usagelicense_organization_id ON LicenseKey (organization_id);

-- Revert ArtifactEntitlement unique constraints
ALTER TABLE ArtifactEntitlement
  DROP CONSTRAINT ArtifactEntitlement_name_unique,
  ADD CONSTRAINT ArtifactLicense_name_unique
    UNIQUE (organization_id, name);

ALTER TABLE ArtifactEntitlement_Artifact
  DROP CONSTRAINT ArtifactEntitlement_Artifact_unique,
  ADD CONSTRAINT ArtifactLicense_Artifact_unique
    UNIQUE NULLS NOT DISTINCT (artifact_entitlement_id, artifact_id, artifact_version_id);

-- Revert ArtifactEntitlement indexes
DROP INDEX IF EXISTS fk_ArtifactEntitlement_Artifact_artifact_version_id;
CREATE INDEX fk_ArtifactLicense_Artifact_artifact_version_id ON ArtifactEntitlement_Artifact (artifact_entitlement_id);

DROP INDEX IF EXISTS fk_ArtifactEntitlement_Artifact_artifact_id;
CREATE INDEX fk_ArtifactLicense_Artifact_artifact_id ON ArtifactEntitlement_Artifact (artifact_id);

DROP INDEX IF EXISTS fk_ArtifactEntitlement_Artifact_artifact_entitlement_id;
CREATE INDEX fk_ArtifactLicense_Artifact_artifact_license_id ON ArtifactEntitlement_Artifact (artifact_entitlement_id);

DROP INDEX IF EXISTS fk_ArtifactEntitlement_customer_organization_id;
CREATE INDEX fk_ArtifactLicense_customer_organization_id ON ArtifactEntitlement (customer_organization_id);

DROP INDEX IF EXISTS fk_ArtifactEntitlement_organization_id;
CREATE INDEX fk_ArtifactLicense_organization_id ON ArtifactEntitlement (organization_id);

-- Revert ApplicationEntitlement indexes
DROP INDEX IF EXISTS fk_ApplicationEntitlement_ApplicationVersion_application_entitlement_id;
CREATE INDEX fk_ApplicationLicense_ApplicationVersion_application_license_id
  ON ApplicationEntitlement_ApplicationVersion (application_entitlement_id);

DROP INDEX IF EXISTS fk_Deployment_application_entitlement_id;
CREATE INDEX fk_Deployment_application_license_id ON Deployment (application_entitlement_id);

DROP INDEX IF EXISTS fk_ApplicationEntitlement_customer_organization_id;
CREATE INDEX fk_ApplicationLicense_customer_organization_id ON ApplicationEntitlement (customer_organization_id);

DROP INDEX IF EXISTS fk_ApplicationEntitlement_organization_id;
CREATE INDEX fk_ApplicationLicense_organization_id ON ApplicationEntitlement (organization_id);

DROP INDEX IF EXISTS fk_ApplicationEntitlement_application_id;
CREATE INDEX fk_ApplicationLicense_application_id ON ApplicationEntitlement (application_id);

-- Revert ArtifactEntitlement FK constraints
ALTER TABLE ArtifactEntitlement
  DROP CONSTRAINT artifactentitlement_organization_id_fkey,
  ADD CONSTRAINT artifactlicense_organization_id_fkey
    FOREIGN KEY (organization_id)
    REFERENCES Organization(id)
    ON DELETE CASCADE;

ALTER TABLE ArtifactEntitlement
  DROP CONSTRAINT artifactentitlement_customer_organization_id_fkey,
  ADD CONSTRAINT artifactlicense_customer_organization_id_fkey
    FOREIGN KEY (customer_organization_id)
    REFERENCES CustomerOrganization(id)
    ON DELETE CASCADE;

-- Revert ApplicationEntitlement FK constraints
ALTER TABLE ApplicationEntitlement
  DROP CONSTRAINT applicationentitlement_customer_organization_id_fkey,
  ADD CONSTRAINT applicationlicense_customer_organization_id_fkey
    FOREIGN KEY (customer_organization_id)
    REFERENCES CustomerOrganization(id)
    ON DELETE CASCADE;

ALTER TABLE Deployment
  DROP CONSTRAINT deployment_application_entitlement_id_fkey,
  ADD CONSTRAINT deployment_application_license_id_fkey
    FOREIGN KEY (application_entitlement_id)
    REFERENCES ApplicationEntitlement(id)
    DEFERRABLE INITIALLY IMMEDIATE;

-- Revert LicenseKey -> UsageLicense
ALTER TABLE LicenseKey RENAME TO UsageLicense;

-- Revert ArtifactEntitlement -> ArtifactLicense
ALTER TABLE ArtifactEntitlement_Artifact RENAME COLUMN artifact_entitlement_id TO artifact_license_id;
ALTER TABLE ArtifactEntitlement_Artifact RENAME TO ArtifactLicense_Artifact;
ALTER TABLE ArtifactEntitlement RENAME TO ArtifactLicense;

-- Revert ApplicationEntitlement -> ApplicationLicense
ALTER TABLE Deployment RENAME COLUMN application_entitlement_id TO application_license_id;
ALTER TABLE ApplicationEntitlement_ApplicationVersion RENAME COLUMN application_entitlement_id TO application_license_id;
ALTER TABLE ApplicationEntitlement_ApplicationVersion RENAME TO ApplicationLicense_ApplicationVersion;
ALTER TABLE ApplicationEntitlement RENAME TO ApplicationLicense;
