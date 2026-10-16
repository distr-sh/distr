import {DeploymentTargetLatestMetrics} from './deployment-target-metrics';

export type NotificationRecordType = 'alert' | 'warning' | 'resolved' | 'update_available';
export type NotificationRecordMetricType = 'cpu' | 'memory' | 'disk';

export interface NotificationRecordDeployment {
  customerOrganizationName?: string;
  deploymentTargetName: string;
  helmReleaseName?: string;
  currentVersionName: string;
}

export interface NotificationRecord {
  id: string;
  createdAt: string;
  deploymentTargetId?: string;
  deploymentTargetName?: string;
  customerOrganizationName?: string;
  alertConfigurationId?: string;
  applicationName?: string;
  applicationVersionName?: string;
  type: NotificationRecordType;
  metricType?: NotificationRecordMetricType;
  diskDevice?: string;
  diskPath?: string;
  deploymentRevisionId?: string;
  deploymentStatusMessage?: string;
  updateNotificationConfigurationId?: string;
  applicationVersionId?: string;
  artifactVersionId?: string;
  artifactName?: string;
  artifactVersionName?: string;
  deployments?: NotificationRecordDeployment[];
  deliveryError: string;
  currentDeploymentTargetMetrics?: DeploymentTargetLatestMetrics;
}
