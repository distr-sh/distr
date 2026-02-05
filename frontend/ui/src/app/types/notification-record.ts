import { DeploymentRevisionStatus } from '@distr-sh/distr-sdk';

export interface NotificationRecord {
  id: string;
  createdAt: string;
  deploymentTargetId?: string;
  deploymentTargetName?: string;
  customerOrganizationName?: string;
  applicationName?: string;
  applicationVersionName?: string;
  message: string;
  currentDeploymentRevisionStatus?: DeploymentRevisionStatus;
}
