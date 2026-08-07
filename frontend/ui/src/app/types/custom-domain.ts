export type CustomDomainType = 'app' | 'registry' | 'customer_portal';

export interface CustomDomain {
  id: string;
  createdAt: string;
  domain: string;
  domainType: CustomDomainType;
  organizationId: string;
  customerOrganizationId?: string;
  // Computed live on every response, never persisted, so dnsCheckedAt is when the server last
  // answered this request, not necessarily a check the admin explicitly triggered.
  dnsVerified: boolean;
  dnsDetail: string;
  dnsCheckedAt: string;
}

export interface CreateCustomDomainRequest {
  domain: string;
  domainType: CustomDomainType;
}
