export type CustomDomainType = 'app' | 'registry' | 'customer_portal';

export interface CustomDomain {
  id: string;
  createdAt: string;
  domain: string;
  domainType: CustomDomainType;
  organizationId: string;
  customerOrganizationId?: string;
}

// The result of a live DNS lookup, requested per domain instead of being part of the domain itself,
// so a slow resolver never delays the domain list. Nothing is persisted, so dnsCheckedAt is when the
// server answered the verification request.
export interface CustomDomainVerification {
  customDomainId: string;
  dnsVerified: boolean;
  dnsDetail: string;
  dnsCheckedAt: string;
}

export interface CreateCustomDomainRequest {
  domain: string;
  domainType: CustomDomainType;
}
