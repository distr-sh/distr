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

export interface UpdateNotificationConfiguration {
  id: string;
  createdAt: string;
  name: string;
  enabled: boolean;
  applications: NotificationApplication[];
  artifacts: NotificationArtifact[];
  recipients: NotificationRecipient[];
}

export interface CreateUpdateNotificationConfigurationRequest {
  customerOrganizationId?: string;
  name: string;
  enabled: boolean;
  applicationIds: string[];
  artifactIds: string[];
  userAccountIds: string[];
}
