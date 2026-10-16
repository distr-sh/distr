ALTER TABLE DeploymentRevision
  ADD COLUMN status_type DEPLOYMENT_STATUS_TYPE,
  ADD COLUMN status_message TEXT,
  ADD COLUMN status_created_at TIMESTAMP,
  ADD COLUMN settled_status_type DEPLOYMENT_STATUS_TYPE,
  ADD COLUMN settled_status_created_at TIMESTAMP,
  ADD CONSTRAINT DeploymentRevision_status_complete CHECK (
    (status_type IS NULL) = (status_created_at IS NULL)
      AND (status_type IS NULL) = (status_message IS NULL)
  ),
  ADD CONSTRAINT DeploymentRevision_settled_status_complete CHECK (
    (settled_status_type IS NULL) = (settled_status_created_at IS NULL)
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

UPDATE DeploymentRevision dr SET
  settled_status_type = drs.type,
  settled_status_created_at = drs.created_at
FROM (
  SELECT DISTINCT ON (deployment_revision_id) deployment_revision_id, type, created_at
  FROM DeploymentRevisionStatus
  WHERE type != 'progressing'
  ORDER BY deployment_revision_id, created_at DESC
) drs
WHERE drs.deployment_revision_id = dr.id;

DROP INDEX idx_notification_record_config_prev_status_created;

ALTER TABLE NotificationRecord
  RENAME COLUMN message TO delivery_error;

ALTER TABLE NotificationRecord
  ADD COLUMN deployment_revision_id UUID REFERENCES DeploymentRevision (id) ON DELETE CASCADE,
  ADD COLUMN deployment_status_message TEXT,
  ADD COLUMN resolved_at TIMESTAMP;

UPDATE NotificationRecord r SET
  deployment_revision_id = drs.deployment_revision_id,
  deployment_status_message = drs.message
FROM DeploymentRevisionStatus drs
WHERE drs.id = r.current_deployment_revision_status_id;

UPDATE NotificationRecord r SET deployment_revision_id = drs.deployment_revision_id
FROM DeploymentRevisionStatus drs
WHERE drs.id = r.previous_deployment_revision_status_id
  AND r.deployment_revision_id IS NULL;

-- A stale warning counts as resolved by the first status the deployment reported after the one it warned about.
-- Warnings without such a status stay open, so the agent's next report sends the recovery notification.
UPDATE NotificationRecord r SET resolved_at = (
  SELECT min(s.created_at)
  FROM DeploymentRevisionStatus s
  JOIN DeploymentRevision sdr ON sdr.id = s.deployment_revision_id
  WHERE sdr.deployment_id = prev_dr.deployment_id
    AND s.created_at > prev.created_at
)
FROM DeploymentRevisionStatus prev
JOIN DeploymentRevision prev_dr ON prev_dr.id = prev.deployment_revision_id
WHERE r.type = 'warning'
  AND prev.id = r.previous_deployment_revision_status_id;

ALTER TABLE NotificationRecord
  DROP COLUMN previous_deployment_revision_status_id,
  DROP COLUMN current_deployment_revision_status_id;

CREATE INDEX fk_NotificationRecord_deployment_revision_id
  ON NotificationRecord (deployment_revision_id);
CREATE INDEX idx_notification_record_open_warning
  ON NotificationRecord (deployment_revision_id, alert_configuration_id)
  WHERE type = 'warning' AND resolved_at IS NULL;

DROP TABLE DeploymentRevisionStatus;
