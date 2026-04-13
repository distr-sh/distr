CREATE TABLE CustomerOrganizationLink (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP DEFAULT now(),
  customer_organization_id UUID NOT NULL REFERENCES CustomerOrganization(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  link TEXT NOT NULL
);
