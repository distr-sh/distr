DROP TABLE IF EXISTS AdvisoryEvent;
DROP TABLE IF EXISTS AdvisoryArtifactVersion;
DROP TABLE IF EXISTS AdvisoryApplicationVersion;
DROP TABLE IF EXISTS AdvisoryTag;
DROP TABLE IF EXISTS AdvisoryReference;
DROP TABLE IF EXISTS Advisory;

DROP TYPE IF EXISTS advisory_event_type;
DROP TYPE IF EXISTS advisory_version_relation;
DROP TYPE IF EXISTS advisory_severity;
DROP TYPE IF EXISTS advisory_status;

-- Postgres cannot drop a value from an enum, so the type is recreated without it.
ALTER TYPE FEATURE RENAME TO FEATURE_OLD;

CREATE TYPE FEATURE AS ENUM (
  'licensing',
  'pre_post_scripts',
  'artifact_version_mutable',
  'vendor_billing',
  'deployment_logs_after',
  'partner_management',
  'custom_domains',
  'custom_emails',
  'custom_oidc_providers'
);

-- Has to happen before the column is converted, which would fail on a value the new type
-- does not have.
UPDATE Organization
  SET features = array_remove(features, 'vulnerabilities')
  WHERE 'vulnerabilities' = any(features);

ALTER TABLE Organization ALTER COLUMN features DROP DEFAULT; -- otherwise the following wouldnt work:
ALTER TABLE Organization
  ALTER COLUMN features TYPE FEATURE[]
    USING (features::text[]::FEATURE[]);
ALTER TABLE Organization
  ALTER COLUMN features SET DEFAULT '{licensing}';

DROP TYPE FEATURE_OLD;
