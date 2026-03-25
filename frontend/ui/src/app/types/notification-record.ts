import {DeploymentRevisionStatus} from '@distr-sh/distr-sdk';
import {DeploymentTargetLatestMetrics} from './deployment-target-metrics';

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
  currentDeploymentTargetMetrics?: DeploymentTargetLatestMetrics;
  metricType?: string;
  diskDevice?: string;
  diskPath?: string;
}
