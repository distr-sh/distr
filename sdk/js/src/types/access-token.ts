import {BaseModel} from './base';
import {UserRole} from './user-account';

export interface AccessToken extends BaseModel {
  expiresAt?: string;
  lastUsedAt?: string;
  label?: string;
  userRole?: UserRole;
}

export interface AccessTokenWithKey extends AccessToken {
  key: string;
}

export interface CreateAccessTokenRequest {
  label?: string;
  expiresAt?: Date;
  userRole?: UserRole;
}
