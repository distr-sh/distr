CREATE TABLE PartnerOrganization (
  id              UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at      TIMESTAMP NOT NULL DEFAULT current_timestamp,
  organization_id UUID      NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  name            TEXT      NOT NULL
);

CREATE INDEX fk_PartnerOrganization_organization_id ON PartnerOrganization(organization_id);

ALTER TABLE CustomerOrganization
  ADD COLUMN partner_organization_id UUID REFERENCES PartnerOrganization(id) ON DELETE SET NULL;

CREATE INDEX fk_CustomerOrganization_partner_organization_id ON CustomerOrganization(partner_organization_id);

ALTER TABLE Organization_UserAccount
  ADD COLUMN partner_organization_id UUID REFERENCES PartnerOrganization(id) ON DELETE CASCADE;

CREATE INDEX fk_Organization_UserAccount_partner_organization_id ON Organization_UserAccount(partner_organization_id);

ALTER TABLE Organization_UserAccount
  ADD CONSTRAINT partner_or_customer_org_exclusive
    CHECK (customer_organization_id IS NULL OR partner_organization_id IS NULL);

ALTER TYPE FEATURE ADD VALUE IF NOT EXISTS 'partner_management';
