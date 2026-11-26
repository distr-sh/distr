import {DeploymentType} from '@distr-sh/distr-sdk';
export type SupportBundleStatus = 'open' | 'resolved';

export interface SupportBundleConfigurationEnvVar {
  name: string;
  redacted: boolean;
}

export interface CreateUpdateSupportBundleConfigurationRequest {
  envVars: SupportBundleConfigurationEnvVar[];
}

export interface SupportBundleKubernetesConfiguration {
  includeConfigMapData: boolean;
  includeSecretData: boolean;
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
  deploymentType: DeploymentType;
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
  createdByCustomerOrganizationId?: string;
  createdByPartnerOrganizationId?: string;
  createdByDeleted?: boolean;
  deploymentType: DeploymentType;
  title: string;
  description?: string;
  status: SupportBundleStatus;
  resourceCount: number;
  uploadedAt?: string;
  commentCount: number;
  lastCommentAt?: string;
  statusChangedByUserName?: string;
  statusChangedByImageUrl?: string;
  statusChangedByCustomerOrganizationId?: string;
  statusChangedByPartnerOrganizationId?: string;
  statusChangedByDeleted?: boolean;
  statusChangedAt?: string;
}

export interface CreateSupportBundleRequest {
  deploymentType: DeploymentType;
  title: string;
  description?: string;
}

export interface CreateSupportBundleResponse extends SupportBundle {
  collectCommand: string;
}

export type SupportBundleResourceContentType = 'text/plain' | 'text/x-log' | 'application/yaml' | 'application/json';

export interface SupportBundleResourceSummary {
  id: string;
  createdAt: string;
  name: string;
  contentType: SupportBundleResourceContentType;
  size?: number;
}

export interface SupportBundleResource extends SupportBundleResourceSummary {
  content: string;
}

export interface SupportBundleComment {
  id: string;
  createdAt: string;
  userAccountId?: string;
  userName?: string;
  userImageUrl?: string;
  userCustomerOrganizationId?: string;
  userPartnerOrganizationId?: string;
  userDeleted?: boolean;
  content: string;
}

export interface SupportBundleDetail extends SupportBundle {
  resources: SupportBundleResourceSummary[];
  comments: SupportBundleComment[];
  collectCommand?: string;
}

export interface CreateSupportBundleCommentRequest {
  content: string;
}

export interface UpdateSupportBundleStatusRequest {
  status: SupportBundleStatus;
}
