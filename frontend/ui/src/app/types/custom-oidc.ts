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

export interface CustomOidcConfigurationsResponse {
  configurations: CustomOidcConfiguration[];
}

export interface CustomOidcConfigurationRequest {
  customDomainId: string;
  // Targets a customer's own provider instead of the caller's own; only meaningful on create.
  customerOrganizationId?: string;
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
