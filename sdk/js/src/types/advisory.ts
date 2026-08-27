import {DeploymentType} from './deployment';

export type AdvisoryStatus = 'triage' | 'draft' | 'published' | 'resolved' | 'canceled';

export type AdvisorySeverity = 'none' | 'low' | 'medium' | 'high' | 'critical';

export type AdvisoryVersionRelation = 'affected' | 'fixed';

export type AdvisoryEventType = 'created' | 'status_changed' | 'edited' | 'comment';

export interface Advisory {
  id: string;
  createdAt: string;
  updatedAt: string;
  /** Only ever sent to the vendor organization that owns the advisory. */
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
  /**
   * Whether the advisory is still a live problem for the requesting customer or partner: a
   * deployment of theirs runs an affected version, or they pulled an affected artifact version
   * without since pulling one that carries the fix. Absent for vendors, who see the status
   * instead.
   */
  affected?: boolean;
}

export interface AdvisoryReference {
  url: string;
  label?: string;
}

export interface AdvisoryApplicationVersion {
  applicationId: string;
  applicationName: string;
  applicationType: DeploymentType;
  /** Absent when the application has no uploaded logo and the icon of its type is used. */
  applicationImageUrl?: string;
  applicationVersionId: string;
  applicationVersionName: string;
  relation: AdvisoryVersionRelation;
}

export interface AdvisoryArtifactVersion {
  artifactId: string;
  artifactName: string;
  artifactVersionId: string;
  /** Name of the marked row, which is a digest when the version was marked by digest. */
  artifactVersionName: string;
  artifactVersionDigest: string;
  /** Tags pointing at the same content, so a version marked by digest is still recognisable. */
  artifactVersionTags: string[];
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
  /** The vendor-internal timeline. Empty for customer and partner users. */
  events: AdvisoryEvent[];
}

/**
 * Where a deployment stands relative to an advisory, derived from the version its current
 * revision runs: still on an affected version, on a version marked as containing the fix, or
 * on a version marked as neither.
 */
export type AdvisoryImpactState = 'affected' | 'fixed' | 'not_affected';

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
   * Defaults to `triage` on create, where issues reported through the API wait to be assessed,
   * and leaves the status untouched when omitted on update.
   */
  status?: AdvisoryStatus;
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

/**
 * Changes the status, the severity, or both, leaving the rest of the advisory as it is.
 * At least one of the two must be given.
 */
export interface PatchAdvisoryRequest {
  status?: AdvisoryStatus;
  severity?: AdvisorySeverity;
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
