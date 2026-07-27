DROP TABLE IF EXISTS VulnerabilityEvent;
DROP TABLE IF EXISTS VulnerabilityArtifactVersion;
DROP TABLE IF EXISTS VulnerabilityApplicationVersion;
DROP TABLE IF EXISTS VulnerabilityTag;
DROP TABLE IF EXISTS VulnerabilityReference;
DROP TABLE IF EXISTS Vulnerability;

DROP TYPE IF EXISTS vulnerability_event_type;
DROP TYPE IF EXISTS vulnerability_version_relation;
DROP TYPE IF EXISTS vulnerability_severity;
DROP TYPE IF EXISTS vulnerability_status;

-- ALTER TYPE FEATURE DROP VALUE is not supported by postgres
