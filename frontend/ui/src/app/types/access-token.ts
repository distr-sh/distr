import {BaseModel, UserRole} from '@distr-sh/distr-sdk';

export type AccessTokenSecretSlot = 1 | 2;

export interface AccessTokenSecret {
  slot: AccessTokenSecretSlot;
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
}

/**
 * Present while the key authenticates on its own, which is the format that predates secrets. Its
 * keyId is the hex encoding the token was issued in, so it differs from the one every other
 * credential of the same token is shown with.
 */
export interface AccessTokenLegacyKey {
  keyId: string;
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
}

export interface AccessToken extends BaseModel {
  keyId: string;
  lastUsedAt?: string;
  label?: string;
  userRole?: UserRole;
  legacyKey?: AccessTokenLegacyKey;
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
