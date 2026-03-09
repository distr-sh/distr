-- Create LicenseKey table
CREATE TABLE LicenseKey (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  name TEXT NOT NULL,
  description TEXT,
  payload JSONB NOT NULL DEFAULT '{}',
  not_before TIMESTAMP NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  organization_id UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  customer_organization_id UUID REFERENCES CustomerOrganization(id) ON DELETE CASCADE,
  UNIQUE (organization_id, name)
);

CREATE INDEX idx_licensekey_organization_id ON LicenseKey (organization_id);
CREATE INDEX idx_licensekey_customer_organization_id ON LicenseKey (customer_organization_id);

-- Rename ApplicationLicense -> ApplicationEntitlement
ALTER TABLE ApplicationLicense RENAME TO ApplicationEntitlement;
ALTER TABLE ApplicationLicense_ApplicationVersion RENAME TO ApplicationEntitlement_ApplicationVersion;
ALTER TABLE ApplicationEntitlement_ApplicationVersion RENAME COLUMN application_license_id TO application_entitlement_id;
ALTER TABLE Deployment RENAME COLUMN application_license_id TO application_entitlement_id;

-- Rename ArtifactLicense -> ArtifactEntitlement
ALTER TABLE ArtifactLicense RENAME TO ArtifactEntitlement;
ALTER TABLE ArtifactLicense_Artifact RENAME TO ArtifactEntitlement_Artifact;
ALTER TABLE ArtifactEntitlement_Artifact RENAME COLUMN artifact_license_id TO artifact_entitlement_id;

-- Rename ApplicationEntitlement FK constraints
ALTER TABLE Deployment
  DROP CONSTRAINT deployment_application_license_id_fkey,
  ADD CONSTRAINT deployment_application_entitlement_id_fkey
    FOREIGN KEY (application_entitlement_id)
    REFERENCES ApplicationEntitlement(id)
    DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE ApplicationEntitlement
  DROP CONSTRAINT applicationlicense_customer_organization_id_fkey,
  ADD CONSTRAINT applicationentitlement_customer_organization_id_fkey
    FOREIGN KEY (customer_organization_id)
    REFERENCES CustomerOrganization(id)
    ON DELETE CASCADE;

-- Rename ArtifactEntitlement FK constraints
ALTER TABLE ArtifactEntitlement
  DROP CONSTRAINT artifactlicense_customer_organization_id_fkey,
  ADD CONSTRAINT artifactentitlement_customer_organization_id_fkey
    FOREIGN KEY (customer_organization_id)
    REFERENCES CustomerOrganization(id)
    ON DELETE CASCADE;

ALTER TABLE ArtifactEntitlement
  DROP CONSTRAINT artifactlicense_organization_id_fkey,
  ADD CONSTRAINT artifactentitlement_organization_id_fkey
    FOREIGN KEY (organization_id)
    REFERENCES Organization(id)
    ON DELETE CASCADE;

-- Rename ApplicationEntitlement indexes
DROP INDEX IF EXISTS fk_ApplicationLicense_application_id;
CREATE INDEX fk_ApplicationEntitlement_application_id ON ApplicationEntitlement (application_id);

DROP INDEX IF EXISTS fk_ApplicationLicense_organization_id;
CREATE INDEX fk_ApplicationEntitlement_organization_id ON ApplicationEntitlement (organization_id);

DROP INDEX IF EXISTS fk_ApplicationLicense_customer_organization_id;
CREATE INDEX fk_ApplicationEntitlement_customer_organization_id ON ApplicationEntitlement (customer_organization_id);

DROP INDEX IF EXISTS fk_Deployment_application_license_id;
CREATE INDEX fk_Deployment_application_entitlement_id ON Deployment (application_entitlement_id);

DROP INDEX IF EXISTS fk_ApplicationLicense_ApplicationVersion_application_license_id;
CREATE INDEX fk_ApplicationEntitlement_ApplicationVersion_application_entitlement_id
  ON ApplicationEntitlement_ApplicationVersion (application_entitlement_id);

-- Rename ArtifactEntitlement indexes
DROP INDEX IF EXISTS fk_ArtifactLicense_organization_id;
CREATE INDEX fk_ArtifactEntitlement_organization_id ON ArtifactEntitlement (organization_id);

DROP INDEX IF EXISTS fk_ArtifactLicense_customer_organization_id;
CREATE INDEX fk_ArtifactEntitlement_customer_organization_id ON ArtifactEntitlement (customer_organization_id);

DROP INDEX IF EXISTS fk_ArtifactLicense_Artifact_artifact_license_id;
CREATE INDEX fk_ArtifactEntitlement_Artifact_artifact_entitlement_id ON ArtifactEntitlement_Artifact (artifact_entitlement_id);

DROP INDEX IF EXISTS fk_ArtifactLicense_Artifact_artifact_id;
CREATE INDEX fk_ArtifactEntitlement_Artifact_artifact_id ON ArtifactEntitlement_Artifact (artifact_id);

DROP INDEX IF EXISTS fk_ArtifactLicense_Artifact_artifact_version_id;
CREATE INDEX fk_ArtifactEntitlement_Artifact_artifact_version_id ON ArtifactEntitlement_Artifact (artifact_version_id);

-- Rename ArtifactEntitlement unique constraint
ALTER TABLE ArtifactEntitlement_Artifact
  DROP CONSTRAINT ArtifactLicense_Artifact_unique,
  ADD CONSTRAINT ArtifactEntitlement_Artifact_unique
    UNIQUE NULLS NOT DISTINCT (artifact_entitlement_id, artifact_id, artifact_version_id);

ALTER TABLE ArtifactEntitlement
  DROP CONSTRAINT ArtifactLicense_name_unique,
  ADD CONSTRAINT ArtifactEntitlement_name_unique
    UNIQUE (organization_id, name);
