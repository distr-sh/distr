import dayjs from 'dayjs';
import {AccessToken, AccessTokenSecret, nextAccessTokenExpiry} from './access-token';

describe('nextAccessTokenExpiry', () => {
  const now = dayjs('2026-06-01T12:00:00Z');
  const past = now.subtract(2, 'day').toISOString();
  const recently = now.subtract(1, 'hour').toISOString();
  const soon = now.add(1, 'hour').toISOString();
  const later = now.add(2, 'day').toISOString();
  const token = (...expirations: (string | undefined)[]): AccessToken => ({
    keyId: 'key',
    secrets: expirations.map((expiresAt, i): AccessTokenSecret => ({
      slot: (i + 1) as AccessTokenSecret['slot'],
      createdAt: past,
      expiresAt,
    })),
  });

  it('should report a secret expiring soon although another one outlives it', () => {
    expect(nextAccessTokenExpiry(token(later, soon), now)).toEqual(soon);
  });

  it('should not let a secret without expiration hide one that expires', () => {
    expect(nextAccessTokenExpiry(token(soon, undefined), now)).toEqual(soon);
  });

  it('should ignore a secret replaced by a rotation', () => {
    expect(nextAccessTokenExpiry(token(past, later), now)).toEqual(later);
    expect(nextAccessTokenExpiry(token(past, undefined), now)).toBeUndefined();
  });

  it('should report when the last credential of an expired token expired', () => {
    expect(nextAccessTokenExpiry(token(past, recently), now)).toEqual(recently);
  });
});
