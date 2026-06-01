import {BaseModel} from './base';

export interface PartnerOrganization extends Required<BaseModel> {
  name: string;
}

export interface PartnerOrganizationWithUsage extends PartnerOrganization {
  userCount: number;
  customerOrganizationCount: number;
}

export interface CreateUpdatePartnerOrganizationRequest {
  name: string;
}

export interface AssignCustomerToPartnerRequest {
  partnerOrganizationId?: string;
}
