export interface SupportBundleConfigurationEnvVar {
  id?: string;
  name: string;
  redacted: boolean;
}

export interface SupportBundleConfiguration {
  id: string;
  createdAt: string;
  envVars: SupportBundleConfigurationEnvVar[];
}

export interface CreateUpdateSupportBundleConfigurationRequest {
  envVars: SupportBundleConfigurationEnvVar[];
}

export interface SupportBundle {
  id: string;
  createdAt: string;
  customerOrganizationId: string;
  customerOrganizationName: string;
  createdByUserAccountId: string;
  createdByUserName: string;
  createdByImageUrl?: string;
  title?: string;
  description?: string;
  status: 'initialized' | 'created' | 'resolved';
  resourceCount: number;
}

export interface CreateSupportBundleRequest {
  title?: string;
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
  userAccountId: string;
  userName: string;
  userImageUrl?: string;
  content: string;
}

export interface SupportBundleDetail extends SupportBundle {
  resources: SupportBundleResource[];
  comments: SupportBundleComment[];
}

export interface CreateSupportBundleCommentRequest {
  content: string;
}

export interface UpdateSupportBundleStatusRequest {
  status: string;
}
