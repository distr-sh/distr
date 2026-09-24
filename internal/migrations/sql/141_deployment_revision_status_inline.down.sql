CREATE TABLE DeploymentRevisionStatus (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP DEFAULT now(),
  deployment_revision_id UUID NOT NULL REFERENCES DeploymentRevision (id) ON DELETE CASCADE,
  type DEPLOYMENT_STATUS_TYPE NOT NULL,
  message TEXT NOT NULL
);

CREATE INDEX fk_DeploymentRevisionStatus_deployment_revision_id
  ON DeploymentRevisionStatus (deployment_revision_id);
CREATE INDEX DeploymentRevisionStatus_created_at
  ON DeploymentRevisionStatus (deployment_revision_id, created_at DESC);

INSERT INTO DeploymentRevisionStatus (deployment_revision_id, type, message, created_at)
SELECT id, status_type, status_message, status_created_at
FROM DeploymentRevision
WHERE status_type IS NOT NULL;

-- The message of the settled status is not stored.
INSERT INTO DeploymentRevisionStatus (deployment_revision_id, type, message, created_at)
SELECT id, settled_status_type, '', settled_status_created_at
FROM DeploymentRevision
WHERE settled_status_type IS NOT NULL
  AND settled_status_created_at IS DISTINCT FROM status_created_at;

ALTER TABLE NotificationRecord
  ADD COLUMN previous_deployment_revision_status_id UUID REFERENCES DeploymentRevisionStatus (id) ON DELETE CASCADE,
  ADD COLUMN current_deployment_revision_status_id UUID REFERENCES DeploymentRevisionStatus (id) ON DELETE CASCADE;

-- Only the latest status per revision survived, so a record referring to an older one keeps a NULL
-- status reference.
UPDATE NotificationRecord r SET previous_deployment_revision_status_id = drs.id
FROM DeploymentRevisionStatus drs
WHERE drs.deployment_revision_id = r.previous_deployment_revision_id
  AND drs.created_at = r.previous_status_created_at;

UPDATE NotificationRecord r SET current_deployment_revision_status_id = drs.id
FROM DeploymentRevisionStatus drs
WHERE drs.deployment_revision_id = r.current_deployment_revision_id
  AND drs.created_at = r.current_status_created_at;

DROP INDEX idx_notification_record_config_prev_status_created;
DROP INDEX fk_NotificationRecord_previous_deployment_revision_id;
DROP INDEX fk_NotificationRecord_current_deployment_revision_id;

ALTER TABLE NotificationRecord
  DROP COLUMN previous_deployment_revision_id,
  DROP COLUMN previous_status_created_at,
  DROP COLUMN current_deployment_revision_id,
  DROP COLUMN current_status_created_at,
  DROP COLUMN current_status_type,
  DROP COLUMN current_status_message;

CREATE INDEX idx_notification_record_config_prev_status_created
  ON NotificationRecord (
    alert_configuration_id,
    previous_deployment_revision_status_id,
    created_at DESC
  );

ALTER TABLE DeploymentRevision
  DROP CONSTRAINT DeploymentRevision_status_complete,
  DROP CONSTRAINT DeploymentRevision_settled_status_complete,
  DROP COLUMN status_type,
  DROP COLUMN status_message,
  DROP COLUMN status_created_at,
  DROP COLUMN settled_status_type,
  DROP COLUMN settled_status_created_at;
