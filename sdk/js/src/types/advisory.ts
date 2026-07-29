export type AdvisoryStatus = 'triage' | 'draft' | 'published' | 'resolved' | 'canceled';

export type AdvisorySeverity = 'none' | 'low' | 'medium' | 'high' | 'critical';

export type AdvisoryVersionRelation = 'affected' | 'fixed';

export type AdvisoryEventType =
  | 'created'
  | 'status_changed'
  | 'edited'
  | 'tags_changed'
  | 'versions_changed'
  | 'reference_added'
  | 'reference_removed'
  | 'comment';

export interface Advisory {
  id: string;
  createdAt: string;
  updatedAt: string;
  createdByUserName?: string;
  createdByImageUrl?: string;
  title: string;
  status: AdvisoryStatus;
  severity: AdvisorySeverity;
  cveId?: string;
  tags: string[];
  affectedVersionCount: number;
  fixedVersionCount: number;
  referenceCount: number;
  publishedAt?: string;
  resolvedAt?: string;
}

export interface AdvisoryReference {
  url: string;
  label?: string;
}

export interface AdvisoryApplicationVersion {
  applicationId: string;
  applicationName: string;
  applicationVersionId: string;
  applicationVersionName: string;
  relation: AdvisoryVersionRelation;
}

export interface AdvisoryArtifactVersion {
  artifactId: string;
  artifactName: string;
  artifactVersionId: string;
  artifactVersionName: string;
  relation: AdvisoryVersionRelation;
}

export interface AdvisoryEvent {
  id: string;
  createdAt: string;
  type: AdvisoryEventType;
  message?: string;
  userName?: string;
  userImageUrl?: string;
}

export interface AdvisoryDetail extends Advisory {
  description: string;
  references: AdvisoryReference[];
  applicationVersions: AdvisoryApplicationVersion[];
  artifactVersions: AdvisoryArtifactVersion[];
  events: AdvisoryEvent[];
}

/**
 * Where a deployment stands relative to an advisory, derived from the version its current
 * revision runs: still on an affected version, on a version marked as containing the fix, or
 * on a version marked as neither.
 */
export type AdvisoryImpactState = 'affected' | 'patched' | 'not_affected';

/**
 * One deployment that has run an affected application version at some point. `applicationVersion*`
 * is the most recent affected version it ran and `lastDeployedAt` is when, while
 * `currentApplicationVersion*` is what it runs today and is what `state` is derived from.
 */
export interface AdvisoryImpactedDeployment {
  customerOrganizationId?: string;
  customerOrganizationName?: string;
  deploymentId: string;
  deploymentTargetId: string;
  deploymentTargetName: string;
  applicationId: string;
  applicationName: string;
  applicationVersionId: string;
  applicationVersionName: string;
  currentApplicationVersionId: string;
  currentApplicationVersionName: string;
  state: AdvisoryImpactState;
  lastDeployedAt: string;
}

export interface AdvisoryImpactedPull {
  customerOrganizationId?: string;
  customerOrganizationName?: string;
  artifactId: string;
  artifactName: string;
  artifactVersionId: string;
  artifactVersionName: string;
  pullCount: number;
  lastPulledAt: string;
}

export interface AdvisoryImpact {
  deployments: AdvisoryImpactedDeployment[];
  pulls: AdvisoryImpactedPull[];
}

export interface CreateUpdateAdvisoryRequest {
  title: string;
  description: string;
  severity: AdvisorySeverity;
  /**
   * Unique per organization, ignoring case. Reusing one that another advisory already
   * carries is rejected with 409 Conflict.
   */
  cveId?: string;
  tags: string[];
  references: AdvisoryReference[];
  affectedApplicationVersionIds: string[];
  fixedApplicationVersionIds: string[];
  affectedArtifactVersionIds: string[];
  fixedArtifactVersionIds: string[];
}

/** The statuses an advisory may be created in. Publishing is always a separate step. */
export type InitialAdvisoryStatus = Extract<AdvisoryStatus, 'triage' | 'draft'>;

export interface CreateAdvisoryRequest extends CreateUpdateAdvisoryRequest {
  /**
   * Status the advisory starts in. Defaults to `triage`, where issues reported through
   * the API wait to be assessed. Pass `draft` for an advisory you are writing yourself.
   */
  status?: InitialAdvisoryStatus;
}

export interface UpdateAdvisoryStatusRequest {
  status: AdvisoryStatus;
}

export interface CreateAdvisoryCommentRequest {
  content: string;
}

/**
 * Filters for the advisory list. Each field matches an advisory having any of the
 * given values; omitting a field, or passing an empty array, disables that filter.
 */
export interface AdvisoryFilter {
  status?: AdvisoryStatus[];
  severity?: AdvisorySeverity[];
  tag?: string[];
}
