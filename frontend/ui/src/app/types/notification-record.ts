import {DeploymentRevisionStatus} from '@distr-sh/distr-sdk';

export interface NotificationRecord {
  id: string;
  createdAt: string;
  deploymentTargetId?: string;
  message: string;
  currentDeploymentRevisionStatus?: DeploymentRevisionStatus;
}
