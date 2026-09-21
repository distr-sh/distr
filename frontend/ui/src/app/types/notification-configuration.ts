import {DeploymentType} from '@distr-sh/distr-sdk';

export interface NotificationRecipient {
  id: string;
  email: string;
  name?: string;
  customerOrganizationId?: string;
  partnerOrganizationId?: string;
}

export interface NotificationApplication {
  id: string;
  name: string;
  type: DeploymentType;
  imageUrl?: string;
}

export interface NotificationArtifact {
  id: string;
  name: string;
  imageUrl?: string;
}

export interface ApplicationNotificationConfiguration {
  id: string;
  createdAt: string;
  name: string;
  enabled: boolean;
  updateAvailableTriggerEnabled: boolean;
  applications: NotificationApplication[];
  recipients: NotificationRecipient[];
}

export interface CreateUpdateApplicationNotificationConfigurationRequest {
  name: string;
  enabled: boolean;
  updateAvailableTriggerEnabled: boolean;
  applicationIds: string[];
  userAccountIds: string[];
}

export interface ArtifactNotificationConfiguration {
  id: string;
  createdAt: string;
  name: string;
  enabled: boolean;
  newVersionTriggerEnabled: boolean;
  artifacts: NotificationArtifact[];
  recipients: NotificationRecipient[];
}

export interface CreateUpdateArtifactNotificationConfigurationRequest {
  name: string;
  enabled: boolean;
  newVersionTriggerEnabled: boolean;
  artifactIds: string[];
  userAccountIds: string[];
}
