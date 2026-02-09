CREATE INDEX idx_notification_record_config_prev_status_created
  ON NotificationRecord (
    deployment_status_notification_configuration_id,
    previous_deployment_revision_status_id,
    created_at DESC
  );

CREATE INDEX idx_notification_record_org_customer_created
  ON NotificationRecord (organization_id, customer_organization_id, created_at DESC);
