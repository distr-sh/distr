import {UserRole} from '@distr-sh/distr-sdk';

export interface CustomOidcConfiguration {
  id: string;
  createdAt: string;
  updatedAt: string;
  customDomainId: string;
  name: string;
  slug: string;
  enabled: boolean;
  issuer: string;
  clientId: string;
  clientSecretSet: boolean;
  scopes: string[];
  pkceEnabled?: boolean;
  spInitiated: boolean;
  createUnknownUsers: boolean;
  defaultUserRole: UserRole;
  allowedEmailDomains: string[];
  callbackUrl: string;
}

export interface OrganizationMember {
  id: string;
  email: string;
  name?: string;
}

export interface CustomOidcConfigurationsResponse {
  configurations: CustomOidcConfiguration[];
  membersWithOtherOrganizations: OrganizationMember[];
}

export interface CustomOidcConfigurationRequest {
  customDomainId: string;
  name: string;
  slug: string;
  enabled: boolean;
  issuer: string;
  clientId: string;
  clientSecret?: string;
  scopes: string[];
  pkceEnabled?: boolean;
  spInitiated: boolean;
  createUnknownUsers: boolean;
  defaultUserRole: UserRole;
  allowedEmailDomains: string[];
}
