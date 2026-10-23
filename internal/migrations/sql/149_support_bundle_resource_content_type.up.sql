CREATE TYPE SUPPORT_BUNDLE_RESOURCE_CONTENT_TYPE AS ENUM ('text/plain', 'text/x-log', 'application/yaml', 'application/json');

ALTER TABLE SupportBundleResource
    ADD COLUMN content_type SUPPORT_BUNDLE_RESOURCE_CONTENT_TYPE NOT NULL DEFAULT 'text/plain';
