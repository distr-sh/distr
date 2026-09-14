import {BaseModel, UserRole} from '@distr-sh/distr-sdk';

export type AccessTokenSecretSlot = 1 | 2;

export interface AccessTokenSecret {
  slot: AccessTokenSecretSlot;
  createdAt: string;
  /**
   * Fixed when the secret is created. A token is kept alive by adding a secret that expires later,
   * not by moving an expiration that something in circulation already relies on.
   */
  expiresAt?: string;
  lastUsedAt?: string;
}

export interface AccessToken extends BaseModel {
  /**
   * The beginning of the token, so that a user can tell which of their tokens a client is
   * configured with. It stays the same when the secrets are rotated, and covers only half of the
   * key, since a token is shown in full exactly once.
   */
  keyId: string;
  /**
   * When the token stops working, which is the last of its secrets to expire, since every secret
   * that is still valid authenticates it. Not settable: an expiration belongs to a secret.
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
  /**
   * The expiration of the secret the token is created with, and a token without one never expires.
   * It cannot be changed afterwards, so a token is kept alive by adding a secret that expires later.
   */
  expiresAt?: Date;
  userRole?: UserRole;
}

/**
 * Carries the only part of a token that is not fixed when it is created. An omitted label is left
 * unchanged and an explicit null clears it.
 */
export interface PatchAccessTokenRequest {
  label?: string | null;
}

export interface CreateAccessTokenSecretRequest {
  expiresAt?: Date;
}
