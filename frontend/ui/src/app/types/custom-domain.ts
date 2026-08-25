export type CustomDomainType = 'app' | 'registry' | 'customer_portal';

export interface CustomDomain {
  id: string;
  createdAt: string;
  domain: string;
  domainType: CustomDomainType;
  organizationId: string;
  customerOrganizationId?: string;
  // Whether the domain is currently used for links, mails, agent manifests and registry URLs. The
  // server decides this, so the rule behind it lives in one place.
  verified: boolean;
  verifiedAt?: string;
  verificationCheckedAt?: string;
  // Why the domain's CNAME record was last found not to point at this instance.
  verificationError?: string;
}

// The stored outcome of the last CNAME check, returned by the check that runs on demand. Inconclusive
// means that check did not complete, in which case the domain keeps the state it had before:
// verificationCheckedAt is the inconclusive attempt, while verifiedAt and verificationError still
// describe the last check that reached a conclusion.
export interface CustomDomainVerification {
  customDomainId: string;
  verified: boolean;
  verifiedAt?: string;
  verificationCheckedAt?: string;
  verificationError?: string;
  inconclusive: boolean;
}

export interface CreateCustomDomainRequest {
  domain: string;
  domainType: CustomDomainType;
}
