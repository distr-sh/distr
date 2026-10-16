import {BaseModel, Named} from './base';
import {DeploymentType, HelmChartType} from './deployment';

/**
 * How the latest version of an application is determined.
 * * 'semver' orders versions by semantic versioning, which every version name must follow.
 * * 'chronological' orders versions by their creation date.
 * * 'legacy' orders by semantic versioning where every name follows it and by creation date
 *   otherwise. It applies to applications created before a strategy existed and cannot be set.
 */
export type VersioningStrategy = 'semver' | 'chronological' | 'legacy';

export interface Application extends BaseModel, Named {
  type: DeploymentType;
  imageId?: string;
  imageUrl?: string;
  versioningStrategy?: VersioningStrategy;
  /** Whether deployments of this application may have automatic updates enabled. */
  allowAutomaticUpdates?: boolean;
  versions?: ApplicationVersion[];
}

export interface ApplicationVersion {
  id?: string;
  name: string;
  linkTemplate?: string;
  createdAt?: string;
  archivedAt?: string;
  applicationId?: string;
  chartType?: HelmChartType;
  chartName?: string;
  chartUrl?: string;
  chartVersion?: string;
  /** The user who created this version. Only vendor users get to see it. */
  createdBy?: ApplicationVersionCreator;
  resources?: ApplicationVersionResource[];
}

export interface ApplicationVersionCreator {
  id: string;
  name?: string;
  email?: string;
  imageId?: string;
}

export interface ApplicationVersionResource {
  id?: string;
  applicationVersionId?: string;
  name: string;
  content: string;
  visibleToCustomers: boolean;
}

export interface PatchApplicationRequest {
  name?: string;
  versioningStrategy?: VersioningStrategy;
  allowAutomaticUpdates?: boolean;
  versions?: {id: string; archivedAt?: string}[];
}
