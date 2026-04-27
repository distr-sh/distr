import {Application} from './application';
import {BaseModel} from './base';

export interface Deployment extends BaseModel {
  deploymentTargetId: string;
  releaseName?: string;
  dockerType?: DockerType;
}

export interface DeploymentRequest {
  deploymentTargetId: string;
  applicationVersionId: string;
  deploymentId?: string;
  applicationEntitlementId?: string;
  releaseName?: string;
  dockerType?: DockerType;
  valuesYaml?: string;
  envFileData?: string;
  forceRestart?: boolean;
  ignoreRevisionSkew?: boolean;
  helmOptions?: HelmOptions;
  tofuVars?: Record<string, unknown>;
  tofuBackendConfig?: Record<string, string>;
}

export interface HelmOptions {
  timeout: string;
  waitStrategy: string;
  rollbackOnFailure: boolean;
  cleanupOnFailure: boolean;
}

export interface DeploymentWithLatestRevision extends Deployment {
  application: Application;
  /**
   * @deprecated Use application.id instead
   */
  applicationId: string;
  /**
   * @deprecated Use application.name instead
   */
  applicationName: string;
  applicationVersionId: string;
  applicationVersionName: string;
  applicationLink: string;
  applicationEntitlementId?: string;
  valuesYaml?: string;
  envFileData?: string;
  deploymentRevisionId?: string;
  deploymentRevisionCreatedAt?: string;
  latestStatus?: DeploymentRevisionStatus;
  helmOptions?: HelmOptions;
}

export interface DeploymentRevisionStatus extends BaseModel {
  type: DeploymentStatusType;
  message: string;
}

export type DeploymentType = 'docker' | 'kubernetes' | 'opentofu';

export type HelmChartType = 'repository' | 'oci';

export type DockerType = 'compose' | 'swarm';

export type DeploymentStatusType = 'healthy' | 'running' | 'progressing' | 'error';

export type DeploymentTargetScope = 'cluster' | 'namespace';
