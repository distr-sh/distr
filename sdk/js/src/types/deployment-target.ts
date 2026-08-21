import {AgentVersion} from './agent-version';
import {BaseModel, Named} from './base';
import {CustomerOrganization} from './customer-organization';
import {DeploymentTargetScope, DeploymentType, DeploymentWithLatestRevision} from './deployment';

export interface DeploymentTarget extends BaseModel, Named {
  name: string;
  type: DeploymentType;
  namespace?: string;
  scope?: DeploymentTargetScope;
  customerOrganization?: CustomerOrganization;
  deployments: DeploymentWithLatestRevision[];
  agentVersion?: AgentVersion;
  reportedAgentVersionId?: string;
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
