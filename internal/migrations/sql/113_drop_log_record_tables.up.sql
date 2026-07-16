-- Deployment and deployment target log records are stored in Loki from now on.
-- Existing records are not migrated.
DROP TABLE DeploymentLogRecord;

DROP TABLE DeploymentTargetLogRecord;
