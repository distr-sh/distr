import {DeploymentTarget, UserAccount} from '@distr-sh/distr-sdk';

export interface CreateUpdateAlertConfigurationRequest {
  name: string;
  enabled: boolean;
  deploymentTargetIds?: string[];
  userAccountIds?: string[];
}

export interface AlertConfiguration {
  id: string;
  createdAt: string;
  name: string;
  enabled: boolean;
  deploymentTargetIds?: string[];
  userAccountIds?: string[];
  userAccounts?: UserAccount[];
  deploymentTargets?: DeploymentTarget[];
}
