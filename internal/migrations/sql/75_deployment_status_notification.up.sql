CREATE TABLE DeploymentStatusNotificationConfiguration (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  organization_id UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  customer_organization_id UUID REFERENCES CustomerOrganization(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE DeploymentStatusNotificationConfiguration_DeploymentTarget (
  deployment_status_notification_configuration_id UUID NOT NULL REFERENCES DeploymentStatusNotificationConfiguration(id) ON DELETE CASCADE,
  deployment_target_id UUID NOT NULL REFERENCES DeploymentTarget(id) ON DELETE CASCADE,
  PRIMARY KEY (deployment_status_notification_configuration_id, deployment_target_id)
);

CREATE TABLE DeploymentStatusNotificationConfiguration_Organization_UserAccount (
  deployment_status_notification_configuration_id UUID NOT NULL REFERENCES DeploymentStatusNotificationConfiguration(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  PRIMARY KEY (deployment_status_notification_configuration_id, organization_id, user_account_id),
  FOREIGN KEY (organization_id, user_account_id) REFERENCES Organization_UserAccount(organization_id, user_account_id) ON DELETE CASCADE
);

CREATE TABLE NotificationRecord (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  organization_id UUID NOT NULL REFERENCES Organization(id) ON DELETE CASCADE,
  customer_organization_id UUID REFERENCES CustomerOrganization(id) ON DELETE CASCADE,
  deployment_target_id UUID REFERENCES DeploymentTarget(id) ON DELETE CASCADE,
  deployment_status_notification_configuration_id UUID REFERENCES DeploymentStatusNotificationConfiguration(id) ON DELETE CASCADE,
  previous_deployment_revision_status_id UUID REFERENCES DeploymentRevisionStatus(id) ON DELETE CASCADE,
  current_deployment_revision_status_id UUID REFERENCES DeploymentRevisionStatus(id) ON DELETE CASCADE,
  message TEXT NOT NULL
);

ALTER TYPE CUSTOMER_ORGANIZATION_FEATURE ADD VALUE IF NOT EXISTS 'notifications';
