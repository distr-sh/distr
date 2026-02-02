export interface CreateUpdateDeploymentStatusNotificationConfigurationRequest {
  name: string;
  enabled: boolean;
  deploymentTargetIds?: string[];
  userAccountIds?: string[];
}

export interface DeploymentStatusNotificationConfiguration {
  id: string;
  createdAt: string;
  name: string;
  enabled: boolean;
  deploymentTargetIds?: string[];
  userAccountIds?: string[];
}
