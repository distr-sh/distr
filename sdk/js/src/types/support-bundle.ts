import {DeploymentType} from './deployment';

export type SupportBundleStatus = 'initialized' | 'created' | 'resolved' | 'canceled';

export interface SupportBundleConfigurationEnvVar {
  name: string;
  redacted: boolean;
}

export interface CreateUpdateSupportBundleConfigurationRequest {
  envVars: SupportBundleConfigurationEnvVar[];
}

export interface SupportBundleConfigurationScript {
  id: string;
  createdAt: string;
  deploymentType: DeploymentType;
  name: string;
  description?: string;
  content: string;
  enabled: boolean;
}

export interface CreateUpdateSupportBundleConfigurationScriptRequest {
  /** Defaults to `docker`. */
  deploymentType?: DeploymentType;
  name: string;
  description?: string;
  content: string;
  enabled: boolean;
}

export interface SupportBundle {
  id: string;
  createdAt: string;
  customerOrganizationId: string;
  customerOrganizationName: string;
  createdByUserAccountId?: string;
  createdByUserName?: string;
  createdByImageUrl?: string;
  deploymentType: DeploymentType;
  title: string;
  description?: string;
  status: SupportBundleStatus;
  resourceCount: number;
  commentCount: number;
  lastCommentAt?: string;
  statusChangedByUserName?: string;
  statusChangedByImageUrl?: string;
  statusChangedAt?: string;
}

export interface CreateSupportBundleRequest {
  /** Defaults to `docker`. */
  deploymentType?: DeploymentType;
  title: string;
  description?: string;
}

export interface CreateSupportBundleResponse extends SupportBundle {
  collectCommand: string;
}

export interface SupportBundleResource {
  id: string;
  createdAt: string;
  name: string;
  content: string;
}

export interface SupportBundleComment {
  id: string;
  createdAt: string;
  userAccountId?: string;
  userName?: string;
  userImageUrl?: string;
  content: string;
}

export interface SupportBundleDetail extends SupportBundle {
  resources: SupportBundleResource[];
  comments: SupportBundleComment[];
  collectCommand?: string;
}

export interface CreateSupportBundleCommentRequest {
  content: string;
}

export interface UpdateSupportBundleStatusRequest {
  status: 'resolved' | 'canceled';
}
