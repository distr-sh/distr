import {BaseModel, UserRole} from '@distr-sh/distr-sdk';

export type AccessTokenSecretSlot = 1 | 2;

export interface AccessTokenSecret {
  slot: AccessTokenSecretSlot;
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
}

export interface AccessToken extends BaseModel {
  keyId: string;
  /**
   * @deprecated Set only for a token that predates secrets. Every other token expires with the
   * secrets that authenticate it.
   */
  expiresAt?: string;
  lastUsedAt?: string;
  label?: string;
  userRole?: UserRole;
  secrets: AccessTokenSecret[];
}

export interface AccessTokenWithKey extends AccessToken {
  key: string;
}

export interface CreateAccessTokenRequest {
  label?: string;
  expiresAt?: Date;
  userRole?: UserRole;
}

export interface PatchAccessTokenRequest {
  label: string;
}

export interface CreateAccessTokenSecretRequest {
  expiresAt?: Date;
}
