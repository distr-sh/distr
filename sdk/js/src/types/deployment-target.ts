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
  currentStatus?: DeploymentTargetStatus;
  deployments: DeploymentWithLatestRevision[];
  agentVersion?: AgentVersion;
  reportedAgentVersionId?: string;
  metricsEnabled: boolean;
  autohealEnabled?: boolean;
  resources?: DeploymentTargetResources;
}

export interface DeploymentTargetResources {
  cpuRequest: string;
  memoryRequest: string;
  cpuLimit: string;
  memoryLimit: string;
}

export interface DeploymentTargetStatus extends BaseModel {
  message: string;
}
