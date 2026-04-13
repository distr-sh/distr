CREATE TABLE SidebarLink (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP DEFAULT now(),
  organization_id UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  customer_organization_id UUID REFERENCES CustomerOrganization(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  link TEXT NOT NULL
);

CREATE INDEX idx_sidebar_link_organization_id ON SidebarLink (organization_id);
CREATE INDEX idx_sidebar_link_customer_organization_id ON SidebarLink (customer_organization_id);
