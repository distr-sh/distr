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
  customerMessage?: string;
  applications: NotificationApplication[];
  recipients: NotificationRecipient[];
}

export interface CreateUpdateApplicationNotificationConfigurationRequest {
  name: string;
  enabled: boolean;
  updateAvailableTriggerEnabled: boolean;
  customerMessage?: string;
  applicationIds: string[];
  userAccountIds: string[];
}

export interface ArtifactNotificationConfiguration {
  id: string;
  createdAt: string;
  name: string;
  enabled: boolean;
  newVersionTriggerEnabled: boolean;
  customerMessage?: string;
  artifacts: NotificationArtifact[];
  recipients: NotificationRecipient[];
}

export interface CreateUpdateArtifactNotificationConfigurationRequest {
  name: string;
  enabled: boolean;
  newVersionTriggerEnabled: boolean;
  customerMessage?: string;
  artifactIds: string[];
  userAccountIds: string[];
}
