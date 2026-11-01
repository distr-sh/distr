import {BaseModel, UserRole} from '@distr-sh/distr-sdk';
import dayjs, {Dayjs} from 'dayjs';

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

/**
 * The next expiration among the credentials of the token that still authenticate, or undefined when none of
 * them expires. Once every credential has expired, the last expiration. A credential that expired while
 * another one still authenticates is ignored as a completed rotation.
 */
export function nextAccessTokenExpiry(token: AccessToken, now: Dayjs = dayjs()): string | undefined {
  const expirations = [...(token.legacyKey ? [token.legacyKey] : []), ...token.secrets].map(
    (credential) => credential.expiresAt
  );
  const valid = expirations.filter((expiresAt) => expiresAt === undefined || now.isBefore(expiresAt));
  if (valid.length > 0) {
    return valid
      .filter((expiresAt) => expiresAt !== undefined)
      .sort((a, b) => dayjs(a).valueOf() - dayjs(b).valueOf())
      .at(0);
  }
  return expirations
    .filter((expiresAt) => expiresAt !== undefined)
    .sort((a, b) => dayjs(b).valueOf() - dayjs(a).valueOf())
    .at(0);
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
