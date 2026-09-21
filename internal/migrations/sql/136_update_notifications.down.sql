-- Application and artifact notifications have no representation in the old columns.
DELETE FROM NotificationRecord WHERE source_type <> 'alert';

DROP INDEX idx_notification_record_user_subject;
DROP INDEX idx_notification_record_user_account_id;
DROP INDEX idx_notification_record_source_previous_status_created;

ALTER TABLE NotificationRecord
    ADD COLUMN deployment_target_id UUID REFERENCES DeploymentTarget (id) ON DELETE CASCADE,
    ADD COLUMN alert_configuration_id UUID REFERENCES AlertConfiguration (id) ON DELETE CASCADE,
    ADD COLUMN previous_deployment_revision_status_id UUID
        REFERENCES DeploymentRevisionStatus (id) ON DELETE CASCADE,
    ADD COLUMN current_deployment_revision_status_id UUID
        REFERENCES DeploymentRevisionStatus (id) ON DELETE CASCADE,
    ADD COLUMN metric_type TEXT,
    ADD COLUMN disk_device TEXT,
    ADD COLUMN disk_path TEXT,
    ADD COLUMN previous_deployment_target_metrics_id UUID
        REFERENCES DeploymentTargetMetrics (id) ON DELETE CASCADE,
    ADD COLUMN current_deployment_target_metrics_id UUID
        REFERENCES DeploymentTargetMetrics (id) ON DELETE CASCADE;

-- The generic columns carry no foreign key, so a record can refer to a row that has been deleted
-- since. Such a reference has no place to go back to and is dropped.
UPDATE NotificationRecord r
SET deployment_target_id = (SELECT dt.id FROM DeploymentTarget dt WHERE dt.id = r.subject_id),
    alert_configuration_id = (
        SELECT c.id FROM AlertConfiguration c WHERE c.id = r.source_configuration_id
    ),
    previous_deployment_revision_status_id = (
        SELECT s.id FROM DeploymentRevisionStatus s
        WHERE s.id = (r.details ->> 'previousDeploymentRevisionStatusId')::UUID
    ),
    current_deployment_revision_status_id = (
        SELECT s.id FROM DeploymentRevisionStatus s
        WHERE s.id = (r.details ->> 'currentDeploymentRevisionStatusId')::UUID
    ),
    previous_deployment_target_metrics_id = (
        SELECT m.id FROM DeploymentTargetMetrics m
        WHERE m.id = (r.details ->> 'previousDeploymentTargetMetricsId')::UUID
    ),
    current_deployment_target_metrics_id = (
        SELECT m.id FROM DeploymentTargetMetrics m
        WHERE m.id = (r.details ->> 'currentDeploymentTargetMetricsId')::UUID
    ),
    metric_type = r.details ->> 'metricType',
    disk_device = r.details ->> 'diskDevice',
    disk_path = r.details ->> 'diskPath';

CREATE INDEX idx_notification_record_config_prev_status_created
    ON NotificationRecord (
        alert_configuration_id,
        previous_deployment_revision_status_id,
        created_at DESC
    );

ALTER TABLE NotificationRecord
    DROP COLUMN user_account_id,
    DROP COLUMN source_type,
    DROP COLUMN source_configuration_id,
    DROP COLUMN subject_id,
    DROP COLUMN details;

DROP TYPE NOTIFICATION_SOURCE_TYPE;

-- ALTER TYPE CUSTOMER_ORGANIZATION_FEATURE DROP VALUE is not supported by postgres, and recreating
-- the type would fail for every value a later migration added that a customer still holds. The
-- value is harmless when unused.
UPDATE CustomerOrganization SET features = array_remove(features, 'update_notifications')
WHERE 'update_notifications' = ANY(features);

DROP TABLE ArtifactNotificationConfiguration_Organization_UserAccount;
DROP TABLE ArtifactNotificationConfiguration_Artifact;
DROP TABLE ArtifactNotificationConfiguration;

DROP TABLE ApplicationNotificationConfiguration_Organization_UserAccount;
DROP TABLE ApplicationNotificationConfiguration_Application;
DROP TABLE ApplicationNotificationConfiguration;
