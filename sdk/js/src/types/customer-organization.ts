import {BaseModel} from './base';

export type CustomerOrganizationFeature = 'deployment_targets' | 'artifacts' | 'alerts' | 'support_bundles';

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

export interface CustomerOrganizationLink {
  id: string;
  createdAt: string;
  customerOrganizationId: string;
  name: string;
  link: string;
}

export interface CustomerOrganizationResponse extends CustomerOrganization {
  links: CustomerOrganizationLink[];
}
