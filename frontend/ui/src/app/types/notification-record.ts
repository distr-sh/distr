import {DeploymentType} from '@distr-sh/distr-sdk';

export type NotificationRecordType = 'alert' | 'warning' | 'resolved' | 'update_available' | 'new_version';
export type NotificationSourceType = 'alert' | 'application' | 'artifact';

export interface NotificationRecordDeployment {
  customerOrganizationName?: string;
  deploymentTargetName: string;
  deploymentName: string;
  currentVersionName?: string;
}

/** Everything specific to what triggered a notification, denormalized when it was sent. */
export interface NotificationRecordDetails {
  summary?: string;
  customerOrganizationName?: string;
  deploymentTargetName?: string;
  applicationName?: string;
  applicationType?: DeploymentType;
  applicationVersionName?: string;
  artifactName?: string;
  artifactVersionName?: string;
  deployments?: NotificationRecordDeployment[];
}

export interface NotificationRecord {
  id: string;
  createdAt: string;
  userAccountId?: string;
  sourceType: NotificationSourceType;
  sourceConfigurationId?: string;
  subjectId?: string;
  type: NotificationRecordType;
  details: NotificationRecordDetails;
  message: string;
}
