import {BaseModel} from './base';
import {UserRole} from './user-account';

export type AccessTokenSecretSlot = 1 | 2;

export interface AccessTokenSecret {
  slot: AccessTokenSecretSlot;
  createdAt: string;
  lastUsedAt?: string;
}

export interface AccessToken extends BaseModel {
  /**
   * The part of the token that identifies it, so that a user can tell which of their tokens a
   * client is configured with. It stays the same when the secrets are rotated.
   */
  keyId: string;
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

/**
 * Supports partial updates: an omitted field is left unchanged, an explicit null clears it, which
 * means no label, no expiry and the role of the user.
 */
export interface PatchAccessTokenRequest {
  label?: string | null;
  expiresAt?: Date | null;
  userRole?: UserRole | null;
}
