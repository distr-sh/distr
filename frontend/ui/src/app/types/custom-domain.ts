export type CustomDomainType = 'app' | 'registry' | 'customer_portal';

export interface CustomDomain {
  id: string;
  createdAt: string;
  domain: string;
  domainType: CustomDomainType;
  organizationId: string;
  customerOrganizationId?: string;
}

export interface CreateCustomDomainRequest {
  domain: string;
  domainType: CustomDomainType;
}
