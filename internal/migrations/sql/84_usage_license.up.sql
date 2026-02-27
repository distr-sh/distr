CREATE TABLE UsageLicense (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  name TEXT NOT NULL,
  description TEXT,
  payload JSONB NOT NULL DEFAULT '{}',
  token TEXT NOT NULL,
  not_before TIMESTAMP NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  organization_id UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  customer_organization_id UUID REFERENCES CustomerOrganization(id) ON DELETE CASCADE,
  UNIQUE (organization_id, name)
);

CREATE INDEX idx_usagelicense_organization_id ON UsageLicense (organization_id);
CREATE INDEX idx_usagelicense_customer_organization_id ON UsageLicense (customer_organization_id);
