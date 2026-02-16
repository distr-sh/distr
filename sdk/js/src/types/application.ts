import {BaseModel, Named} from './base';
import {DeploymentType, HelmChartType} from './deployment';

export interface Application extends BaseModel, Named {
  type: DeploymentType;
  imageId?: string;
  imageUrl?: string;
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
  resources?: ApplicationVersionResource[];
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
  versions?: {id: string; archivedAt?: string}[];
}
