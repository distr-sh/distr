CREATE TABLE Environment (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  organization_id UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
  is_production BOOLEAN NOT NULL DEFAULT false,
  allow_dynamic_targets BOOLEAN NOT NULL DEFAULT false,
  retention_policy_id UUID,
  CONSTRAINT environment_organization_name_unique UNIQUE (organization_id, name)
);

CREATE INDEX Environment_organization_sort_name
  ON Environment (organization_id, sort_order, name);
