import {Application} from './application';
import {BaseModel} from './base';

export interface Deployment extends BaseModel {
  deploymentTargetId: string;
  releaseName?: string;
  dockerType?: DockerType;
  /**
   * Whether this deployment is rolled forward to the application's latest version automatically.
   * Unrelated to DeploymentTarget.automaticUpdatesEnabled, which is the agent updating itself.
   */
  automaticApplicationUpdatesEnabled?: boolean;
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
  /** Leaves an existing deployment's setting alone when absent. */
  automaticApplicationUpdatesEnabled?: boolean;
}

export interface PatchDeploymentRequest {
  automaticApplicationUpdatesEnabled?: boolean;
}

export interface HelmOptions {
  timeout: string;
  waitStrategy: string;
  rollbackOnFailure: boolean;
  cleanupOnFailure: boolean;
  forceConflicts: boolean;
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
  /**
   * The revision an agent last reported as applied, which differs from deploymentRevisionId while a
   * newer revision is being rolled out or has failed.
   */
  currentDeploymentRevisionId?: string;
  currentStatus?: DeploymentRevisionStatus;
  currentApplicationVersionId?: string;
  currentApplicationVersionName?: string;
  helmOptions?: HelmOptions;
}

export interface DeploymentRevisionStatus extends BaseModel {
  type: DeploymentStatusType;
  message: string;
}

export interface DeploymentRevisionCreator {
  id?: string;
  name?: string;
  email?: string;
  imageId?: string;
  customerOrganizationId?: string;
  partnerOrganizationId?: string;
  deleted?: boolean;
}

export interface DeploymentRevisionResponse {
  id: string;
  createdAt: string;
  applicationVersionId: string;
  applicationVersionName: string;
  releaseName?: string;
  dockerType?: DockerType;
  valuesYaml?: string;
  envFileData?: string;
  forceRestart: boolean;
  ignoreRevisionSkew: boolean;
  helmOptions?: HelmOptions;
  trigger: DeploymentRevisionTrigger;
  createdBy?: DeploymentRevisionCreator;
}

export type DeploymentRevisionTrigger = 'user' | 'automatic_update' | 'secret_change' | 'license_key_change';

export type DeploymentType = 'docker' | 'kubernetes';

export type HelmChartType = 'repository' | 'oci';

export type DockerType = 'compose' | 'swarm';

export type DeploymentStatusType = 'healthy' | 'running' | 'progressing' | 'error';

export type DeploymentTargetScope = 'cluster' | 'namespace';
