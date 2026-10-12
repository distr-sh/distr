import {DeploymentRevisionStatus, DeploymentType} from '@distr-sh/distr-sdk';
import {SupportBundleStatus} from './support-bundle';

export interface OverviewUser {
  name: string;
  imageUrl?: string;
}

export interface OverviewDeploymentTarget {
  id: string;
  name: string;
  type: DeploymentType;
  connected: boolean;
  deployments: OverviewDeployment[];
}

export interface OverviewDeployment {
  id: string;
  applicationId: string;
  applicationName: string;
  applicationImageUrl?: string;
  applicationTitleImageUrl?: string;
  applicationLink?: string;
  applicationVersionId: string;
  applicationVersionName: string;
  currentApplicationVersionId?: string;
  currentApplicationVersionName?: string;
  latestApplicationVersionName?: string;
  updateAvailable: boolean;
  deploymentRevisionId: string;
  deploymentRevisionCreatedAt: string;
  deploymentRevisionCreatedBy?: OverviewUser;
  currentDeploymentRevisionId?: string;
  latestStatus?: DeploymentRevisionStatus;
  currentStatus?: DeploymentRevisionStatus;
}

export interface OverviewArtifacts {
  artifacts: OverviewArtifact[];
  /** Also true when only artifacts hidden in the overview were pulled. */
  hasPulled: boolean;
}

export interface OverviewArtifact {
  id: string;
  name: string;
  imageUrl?: string;
  titleImageUrl?: string;
  latestVersionName?: string;
  lastPull?: OverviewArtifactPull;
}

export interface OverviewArtifactPull {
  versionName: string;
  createdAt: string;
  pulledBy?: OverviewUser;
  deploymentTargetName?: string;
  isLatest: boolean;
}

export interface OverviewChecklist {
  profileCompleted: boolean;
  teamMembersInvited: boolean;
  deploymentCreated: boolean;
  secretCreated: boolean;
  accessTokenCreated: boolean;
  artifactPulled: boolean;
}

export interface OverviewLicense {
  name: string;
  notBefore?: string;
  expiresAt?: string;
}

export interface OverviewAlerts {
  alertConfigurationCount: number;
  notificationCount: number;
}

export interface OverviewAccessToken {
  id: string;
  label?: string;
  expiresAt: string;
}

export interface OverviewSupportBundle {
  id: string;
  title: string;
  status: SupportBundleStatus;
  updatedAt: string;
  updatedBy?: OverviewUser;
}

export type OverviewActivityType =
  | 'application_version_created'
  | 'artifact_version_created'
  | 'artifact_version_first_pulled'
  | 'deployment_revision_created'
  | 'license_key_created'
  | 'license_key_updated'
  | 'support_bundle_created'
  | 'support_bundle_uploaded'
  | 'support_bundle_resolved'
  | 'support_bundle_canceled';

export interface OverviewActivity {
  id: string;
  createdAt: string;
  type: OverviewActivityType;
  resourceId: string;
  subject: string;
  detail?: string;
  user?: OverviewUser;
}
