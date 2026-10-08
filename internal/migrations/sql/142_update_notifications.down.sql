DELETE FROM NotificationRecord WHERE update_notification_configuration_id IS NOT NULL;

DROP INDEX idx_notification_record_update_audience;
DROP INDEX fk_NotificationRecord_partner_organization_id;
DROP INDEX fk_NotificationRecord_artifact_version_id;
DROP INDEX fk_NotificationRecord_application_version_id;

ALTER TABLE NotificationRecord
    DROP CONSTRAINT notification_record_update_subject,
    DROP COLUMN deployments,
    DROP COLUMN partner_organization_id,
    DROP COLUMN artifact_version_id,
    DROP COLUMN application_version_id,
    DROP COLUMN update_notification_configuration_id;

-- ALTER TYPE ... DROP VALUE is not supported by postgres, and recreating a type would fail for
-- every value a later migration added. Both values are harmless when unused.
UPDATE CustomerOrganization SET features = array_remove(features, 'update_notifications')
WHERE 'update_notifications' = ANY(features);

DROP TABLE UpdateNotificationConfiguration_Organization_UserAccount;
DROP TABLE UpdateNotificationConfiguration_Artifact;
DROP TABLE UpdateNotificationConfiguration_Application;
DROP TABLE UpdateNotificationConfiguration;
