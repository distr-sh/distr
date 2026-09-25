CREATE TABLE UpdateNotificationConfiguration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    customer_organization_id UUID REFERENCES CustomerOrganization (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX idx_update_notification_configuration_organization_id
    ON UpdateNotificationConfiguration (organization_id);

CREATE TABLE UpdateNotificationConfiguration_Application (
    update_notification_configuration_id UUID NOT NULL
        REFERENCES UpdateNotificationConfiguration (id) ON DELETE CASCADE,
    application_id UUID NOT NULL REFERENCES Application (id) ON DELETE CASCADE,
    PRIMARY KEY (update_notification_configuration_id, application_id)
);

CREATE INDEX idx_update_notification_configuration_application_id
    ON UpdateNotificationConfiguration_Application (application_id);

CREATE TABLE UpdateNotificationConfiguration_Artifact (
    update_notification_configuration_id UUID NOT NULL
        REFERENCES UpdateNotificationConfiguration (id) ON DELETE CASCADE,
    artifact_id UUID NOT NULL REFERENCES Artifact (id) ON DELETE CASCADE,
    PRIMARY KEY (update_notification_configuration_id, artifact_id)
);

CREATE INDEX idx_update_notification_configuration_artifact_id
    ON UpdateNotificationConfiguration_Artifact (artifact_id);

CREATE TABLE UpdateNotificationConfiguration_Organization_UserAccount (
    update_notification_configuration_id UUID NOT NULL
        REFERENCES UpdateNotificationConfiguration (id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    user_account_id UUID NOT NULL REFERENCES UserAccount (id) ON DELETE CASCADE,
    PRIMARY KEY (update_notification_configuration_id, organization_id, user_account_id),
    FOREIGN KEY (organization_id, user_account_id)
        REFERENCES Organization_UserAccount (organization_id, user_account_id) ON DELETE CASCADE
);

ALTER TYPE CUSTOMER_ORGANIZATION_FEATURE ADD VALUE IF NOT EXISTS 'update_notifications';

CREATE TYPE NOTIFICATION_SOURCE_TYPE AS ENUM ('alert', 'application', 'artifact');

ALTER TYPE NOTIFICATION_RECORD_TYPE ADD VALUE IF NOT EXISTS 'update_available';
ALTER TYPE NOTIFICATION_RECORD_TYPE ADD VALUE IF NOT EXISTS 'new_version';

-- source_configuration_id and subject_id are polymorphic and therefore carry no foreign key: a
-- record of what has been sent has to survive the deletion of the configuration, deployment target
-- or version it refers to, which is why details also keeps the names needed to display the row.
ALTER TABLE NotificationRecord
    ADD COLUMN user_account_id UUID REFERENCES UserAccount (id) ON DELETE SET NULL,
    ADD COLUMN source_type NOTIFICATION_SOURCE_TYPE NOT NULL DEFAULT 'alert',
    ADD COLUMN source_configuration_id UUID,
    ADD COLUMN subject_id UUID,
    ADD COLUMN details JSONB NOT NULL DEFAULT '{}'::JSONB;

UPDATE NotificationRecord r
SET source_configuration_id = src.alert_configuration_id,
    subject_id = src.deployment_target_id,
    details = jsonb_strip_nulls(jsonb_build_object(
        'summary', CASE
            WHEN src.metric_type = 'cpu' THEN format('CPU utilization is %s%%', round(src.cpu_usage * 100))
            WHEN src.metric_type = 'memory' THEN format('Memory utilization is %s%%', round(src.memory_usage * 100))
            WHEN src.metric_type = 'disk' AND coalesce(src.bytes_total, 0) > 0
                THEN format('Disk utilization is %s%%', round(src.bytes_used::NUMERIC / src.bytes_total * 100))
            WHEN src.metric_type = 'disk' THEN 'Disk utilization is N/A'
            WHEN src.type = 'warning' THEN 'Stale'
            ELSE src.deployment_status_message
        END,
        'deploymentTargetName', src.deployment_target_name,
        'customerOrganizationName', src.customer_organization_name,
        'applicationName', src.application_name,
        'applicationType', src.application_type,
        'applicationVersionName', src.application_version_name,
        'metricType', src.metric_type,
        'diskDevice', src.disk_device,
        'diskPath', src.disk_path,
        'previousDeploymentTargetMetricsId', src.previous_deployment_target_metrics_id,
        'currentDeploymentTargetMetricsId', src.current_deployment_target_metrics_id
    ))
FROM (
    SELECT nr.id,
        nr.type,
        nr.alert_configuration_id,
        nr.deployment_target_id,
        nr.deployment_status_message,
        nr.metric_type,
        nr.disk_device,
        nr.disk_path,
        nr.previous_deployment_target_metrics_id,
        nr.current_deployment_target_metrics_id,
        dt.name AS deployment_target_name,
        co.name AS customer_organization_name,
        a.name AS application_name,
        a.type AS application_type,
        av.name AS application_version_name,
        dtm.cpu_usage,
        dtm.memory_usage,
        dtdm.bytes_used,
        dtdm.bytes_total
    FROM NotificationRecord nr
    LEFT JOIN DeploymentTarget dt ON dt.id = nr.deployment_target_id
    LEFT JOIN CustomerOrganization co ON co.id = dt.customer_organization_id
    LEFT JOIN DeploymentRevision dr ON dr.id = nr.deployment_revision_id
    LEFT JOIN ApplicationVersion av ON av.id = dr.application_version_id
    LEFT JOIN Application a ON a.id = av.application_id
    LEFT JOIN DeploymentTargetMetrics dtm ON dtm.id = nr.current_deployment_target_metrics_id
    LEFT JOIN DeploymentTargetDiskMetrics dtdm
        ON dtdm.deployment_target_metrics_id = dtm.id
            AND dtdm.device = nr.disk_device
            AND dtdm.path = nr.disk_path
) src
WHERE src.id = r.id;

DROP INDEX idx_notification_record_open_warning;

ALTER TABLE NotificationRecord
    DROP COLUMN deployment_target_id,
    DROP COLUMN alert_configuration_id,
    DROP COLUMN deployment_status_message,
    DROP COLUMN metric_type,
    DROP COLUMN disk_device,
    DROP COLUMN disk_path,
    DROP COLUMN previous_deployment_target_metrics_id,
    DROP COLUMN current_deployment_target_metrics_id,
    ALTER COLUMN source_type DROP DEFAULT;

-- Deleting a deployment keeps the records about it for the same reason the generic columns carry
-- no foreign key. A warning left without its revision can no longer be matched to a deployment,
-- so it neither suppresses nor resolves anything.
ALTER TABLE NotificationRecord
    DROP CONSTRAINT notificationrecord_deployment_revision_id_fkey,
    ADD CONSTRAINT notificationrecord_deployment_revision_id_fkey
        FOREIGN KEY (deployment_revision_id) REFERENCES DeploymentRevision (id) ON DELETE SET NULL;

CREATE INDEX idx_notification_record_open_warning
    ON NotificationRecord (deployment_revision_id, source_configuration_id)
    WHERE type = 'warning' AND resolved_at IS NULL;

-- Deleting a user account nulls the column, which without an index scans the whole table.
CREATE INDEX idx_notification_record_user_account_id
    ON NotificationRecord (user_account_id);

-- A recipient hears about a version once, whichever configuration reaches them first, no matter
-- how many deployments of it are affected, whether a mutable tag is pushed again, or whether an
-- entitlement change replays the notification. Alert records carry no user account and stay
-- outside this index.
CREATE UNIQUE INDEX idx_notification_record_user_subject
    ON NotificationRecord (user_account_id, subject_id)
    WHERE user_account_id IS NOT NULL AND subject_id IS NOT NULL;
