UPDATE Organization
  SET features = array_remove(features, 'vulnerabilities')
  WHERE 'vulnerabilities' = any(features);

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

-- ALTER TYPE FEATURE DROP VALUE is not supported by postgres
