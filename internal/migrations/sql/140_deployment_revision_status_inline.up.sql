ALTER TABLE DeploymentRevision
  ADD COLUMN status_type DEPLOYMENT_STATUS_TYPE,
  ADD COLUMN status_message TEXT,
  ADD COLUMN status_created_at TIMESTAMP,
  ADD CONSTRAINT DeploymentRevision_status_complete CHECK (
    (status_type IS NULL) = (status_created_at IS NULL)
      AND (status_type IS NULL) = (status_message IS NULL)
  );

UPDATE DeploymentRevision dr SET
  status_type = drs.type,
  status_message = drs.message,
  status_created_at = drs.created_at
FROM (
  SELECT DISTINCT ON (deployment_revision_id) deployment_revision_id, type, message, created_at
  FROM DeploymentRevisionStatus
  ORDER BY deployment_revision_id, created_at DESC
) drs
WHERE drs.deployment_revision_id = dr.id;

DROP INDEX idx_notification_record_config_prev_status_created;

ALTER TABLE NotificationRecord
  ADD COLUMN previous_deployment_revision_id UUID REFERENCES DeploymentRevision (id) ON DELETE CASCADE,
  ADD COLUMN previous_status_created_at TIMESTAMP,
  ADD COLUMN current_deployment_revision_id UUID REFERENCES DeploymentRevision (id) ON DELETE CASCADE,
  ADD COLUMN current_status_created_at TIMESTAMP,
  ADD COLUMN current_status_type DEPLOYMENT_STATUS_TYPE,
  ADD COLUMN current_status_message TEXT;

UPDATE NotificationRecord r SET
  previous_deployment_revision_id = drs.deployment_revision_id,
  previous_status_created_at = drs.created_at
FROM DeploymentRevisionStatus drs
WHERE drs.id = r.previous_deployment_revision_status_id;

UPDATE NotificationRecord r SET
  current_deployment_revision_id = drs.deployment_revision_id,
  current_status_created_at = drs.created_at,
  current_status_type = drs.type,
  current_status_message = drs.message
FROM DeploymentRevisionStatus drs
WHERE drs.id = r.current_deployment_revision_status_id;

ALTER TABLE NotificationRecord
  DROP COLUMN previous_deployment_revision_status_id,
  DROP COLUMN current_deployment_revision_status_id;

CREATE INDEX idx_notification_record_config_prev_status_created
  ON NotificationRecord (
    alert_configuration_id,
    previous_deployment_revision_id,
    previous_status_created_at,
    created_at DESC
  );

CREATE INDEX fk_NotificationRecord_previous_deployment_revision_id
  ON NotificationRecord (previous_deployment_revision_id);
CREATE INDEX fk_NotificationRecord_current_deployment_revision_id
  ON NotificationRecord (current_deployment_revision_id);

DROP TABLE DeploymentRevisionStatus;
