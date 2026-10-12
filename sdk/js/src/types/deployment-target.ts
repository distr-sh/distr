import {BaseModel, Named} from './base';
import {ControllerVersion} from './controller-version';
import {CustomerOrganization} from './customer-organization';
import {DeploymentTargetScope, DeploymentType, DeploymentWithLatestRevision} from './deployment';

export interface DeploymentTarget extends BaseModel, Named {
  name: string;
  type: DeploymentType;
  namespace?: string;
  scope?: DeploymentTargetScope;
  customerOrganization?: CustomerOrganization;
  deployments: DeploymentWithLatestRevision[];
  controllerVersion?: ControllerVersion;
  reportedControllerVersionId?: string;
  reconnectPending?: boolean;
  metricsEnabled: boolean;
  imageCleanupEnabled: boolean;
  deploymentLogsEnabled: boolean;
  deploymentLogsAfter?: string;
  autohealEnabled?: boolean;
  automaticUpdatesEnabled?: boolean;
  resources?: DeploymentTargetResources;
  dockerEndpoint?: string;
}

export interface DeploymentTargetResources {
  cpuRequest: string;
  memoryRequest: string;
  cpuLimit: string;
  memoryLimit: string;
}

export interface DeploymentTargetSettings {
  name: string;
  metricsEnabled: boolean;
  imageCleanupEnabled: boolean;
  deploymentLogsEnabled: boolean;
  deploymentLogsAfter?: string;
  automaticUpdatesEnabled?: boolean;
  resources?: DeploymentTargetResources;
  dockerEndpoint?: string;
}

export interface CreateDeploymentTargetRequest extends DeploymentTargetSettings {
  type: DeploymentType;
  namespace?: string;
  scope?: DeploymentTargetScope;
  autohealEnabled?: boolean;
  customerOrganizationId?: string;
}

export interface UpdateDeploymentTargetRequest extends DeploymentTargetSettings {
  controllerVersionId?: string;
}
