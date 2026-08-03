export type CustomDomainType = 'app' | 'registry';

export interface CustomDomain {
  id: string;
  createdAt: string;
  domain: string;
  domainType: CustomDomainType;
  organizationId: string;
}

export interface CreateCustomDomainRequest {
  domain: string;
  domainType: CustomDomainType;
}
