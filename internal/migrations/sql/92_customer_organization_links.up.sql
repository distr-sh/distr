CREATE TABLE CustomerOrganizationLink (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP DEFAULT now(),
  customer_organization_id UUID NOT NULL REFERENCES CustomerOrganization(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  link TEXT NOT NULL
);

CREATE INDEX idx_customer_organization_link_customer_organization_id ON CustomerOrganizationLink (customer_organization_id);
