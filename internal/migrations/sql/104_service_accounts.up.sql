CREATE TABLE ServiceAccount (
  id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at               TIMESTAMP NOT NULL DEFAULT current_timestamp,
  organization_id          UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  customer_organization_id UUID REFERENCES CustomerOrganization(id) ON DELETE CASCADE,
  name                     TEXT NOT NULL,
  account_role             ACCOUNT_ROLE NOT NULL,
  UNIQUE (organization_id, customer_organization_id, name)
);

CREATE INDEX fk_ServiceAccount_organization_id ON ServiceAccount(organization_id);
CREATE INDEX fk_ServiceAccount_customer_organization_id ON ServiceAccount(customer_organization_id);

CREATE TABLE ServiceAccountAccessToken (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at         TIMESTAMP NOT NULL DEFAULT current_timestamp,
  expires_at         TIMESTAMP,
  last_used_at       TIMESTAMP,
  label              TEXT,
  key                BYTEA UNIQUE NOT NULL,
  service_account_id UUID NOT NULL REFERENCES ServiceAccount(id) ON DELETE CASCADE
);

CREATE INDEX ServiceAccountAccessToken_key ON ServiceAccountAccessToken(key);
CREATE INDEX fk_ServiceAccountAccessToken_service_account_id ON ServiceAccountAccessToken(service_account_id);
