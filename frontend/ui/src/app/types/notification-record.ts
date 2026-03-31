import {DeploymentRevisionStatus} from '@distr-sh/distr-sdk';
import {DeploymentTargetLatestMetrics} from './deployment-target-metrics';

export type NotificationRecordType = 'alert' | 'warning' | 'resolved';
export type NotificationRecordMetricType = 'cpu' | 'memory' | 'disk';

export interface NotificationRecord {
  id: string;
  createdAt: string;
  deploymentTargetId?: string;
  deploymentTargetName?: string;
  customerOrganizationName?: string;
  applicationName?: string;
  applicationVersionName?: string;
  type: NotificationRecordType;
  metricType?: NotificationRecordMetricType;
  diskDevice?: string;
  diskPath?: string;
  message: string;
  currentDeploymentRevisionStatus?: DeploymentRevisionStatus;
  currentDeploymentTargetMetrics?: DeploymentTargetLatestMetrics;
}
