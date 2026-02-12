CREATE TABLE ApplicationVersionResource (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  application_version_id UUID NOT NULL REFERENCES ApplicationVersion (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  content TEXT NOT NULL,
  visible_to_customers BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_application_version_resource_version_id
  ON ApplicationVersionResource (application_version_id);
