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

-- Records do not store which status they refer to, so the status references stay NULL.
DROP INDEX fk_NotificationRecord_deployment_revision_id;
DROP INDEX idx_notification_record_open_warning;

ALTER TABLE NotificationRecord
  DROP COLUMN deployment_revision_id,
  DROP COLUMN deployment_status_message,
  DROP COLUMN resolved_at;

ALTER TABLE NotificationRecord
  RENAME COLUMN delivery_error TO message;

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
