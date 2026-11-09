import {DeploymentRevisionStatus, DeploymentType} from '@distr-sh/distr-sdk';
import {ActivityFeedEntryType} from '../../types/activity-feed';
import {SupportBundleStatus} from '../../types/support-bundle';

export interface CustomerHomeUser {
  name: string;
  imageUrl?: string;
}

export interface CustomerHomeDeploymentTarget {
  id: string;
  name: string;
  type: DeploymentType;
  connected: boolean;
  deployments: CustomerHomeDeployment[];
}

export interface CustomerHomeDeployment {
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
  deploymentRevisionId?: string;
  deploymentRevisionCreatedAt?: string;
  currentDeploymentRevisionId?: string;
  latestStatus?: DeploymentRevisionStatus;
  currentStatus?: DeploymentRevisionStatus;
}

export interface CustomerHomeArtifacts {
  artifacts: CustomerHomeArtifact[];
  /** Also true when only artifacts hidden on the home page were pulled. */
  hasPulled: boolean;
}

export interface CustomerHomeArtifact {
  id: string;
  name: string;
  imageUrl?: string;
  titleImageUrl?: string;
  lastPull?: CustomerHomeArtifactPull;
}

export interface CustomerHomeArtifactPull {
  versionName: string;
  createdAt: string;
  pulledBy?: CustomerHomeUser;
  deploymentTargetName?: string;
}

export interface CustomerHomeChecklist {
  profileCompleted: boolean;
  teamMembersInvited: boolean;
  deploymentCreated: boolean;
  secretCreated: boolean;
  accessTokenCreated: boolean;
  artifactPulled: boolean;
  /** Undefined when the customer has no update notifications. */
  updateNotificationsConfigured?: boolean;
}

export interface CustomerHomeLicense {
  name: string;
  notBefore?: string;
  expiresAt?: string;
  entitlement: boolean;
  /** False when the customer has no license key, so the license keys page has nothing to show. */
  hasLicenseKeys: boolean;
}

export interface CustomerHomeAlerts {
  alertConfigurationCount: number;
  notificationCount: number;
}
export interface CustomerHomeAccessToken {
  id: string;
  label?: string;
  expiresAt: string;
  lastUsedAt?: string;
}

export interface CustomerHomeSupportBundle {
  id: string;
  title: string;
  status: SupportBundleStatus;
  updatedAt: string;
  updatedBy?: CustomerHomeUser;
}

export interface CustomerHomeActivity {
  id: string;
  createdAt: string;
  type: ActivityFeedEntryType;
  resourceId: string;
  subject: string;
  detail?: string;
  user?: CustomerHomeUser;
}
