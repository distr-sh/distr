import {UserRole} from '@distr-sh/distr-sdk';

export interface CustomOidcConfiguration {
  id: string;
  createdAt: string;
  updatedAt: string;
  customDomainId: string;
  name: string;
  enabled: boolean;
  issuer: string;
  clientId: string;
  // Whether a client secret is stored. The secret itself is never returned.
  clientSecretSet: boolean;
  scopes: string[];
  pkceEnabled?: boolean;
  spInitiated: boolean;
  createUnknownUsers: boolean;
  defaultUserRole: UserRole;
  allowedEmailDomains: string[];
  // The redirect URI that has to be registered with the identity provider.
  callbackUrl: string;
}

export interface CustomOidcConfigurationsResponse {
  configurations: CustomOidcConfiguration[];
  // Members that are also members of another organization and therefore cannot use any provider.
  membersWithOtherOrganizations: number;
}

export interface CustomOidcConfigurationRequest {
  customDomainId: string;
  name: string;
  enabled: boolean;
  issuer: string;
  clientId: string;
  // Omitted keeps the stored secret.
  clientSecret?: string;
  scopes: string[];
  pkceEnabled?: boolean;
  spInitiated: boolean;
  createUnknownUsers: boolean;
  defaultUserRole: UserRole;
  allowedEmailDomains: string[];
}
