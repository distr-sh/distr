import {BaseModel} from './base';

export type CustomerOrganizationFeature = 'deployment_targets' | 'artifacts' | 'notifications';

export interface CustomerOrganization extends Required<BaseModel> {
  name: string;
  imageId?: string;
  imageUrl?: string;
  features: CustomerOrganizationFeature[];
}

export interface CustomerOrganizationWithUsage extends CustomerOrganization {
  userCount: number;
  deploymentTargetCount: number;
}

export interface CreateUpdateCustomerOrganizationRequest {
  name: string;
  imageId?: string;
  features?: CustomerOrganizationFeature[];
}
