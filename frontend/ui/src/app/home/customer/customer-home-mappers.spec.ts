import dayjs from 'dayjs';
import {AccessToken, AccessTokenSecret} from '../../types/access-token';
import {LicenseKey} from '../../types/license-key';
import {nextAccessTokenExpiry, toCustomerHomeLicense} from './customer-home-mappers';

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

describe('toCustomerHomeLicense', () => {
  const now = dayjs('2026-06-01T12:00:00Z');
  const key = (name: string, notBefore: dayjs.Dayjs, expiresAt: dayjs.Dayjs): LicenseKey => ({
    name,
    notBefore: notBefore.toISOString(),
    expiresAt: expiresAt.toISOString(),
  });
  const valid = key('valid', now.subtract(1, 'year'), now.add(1, 'month'));
  const renewal = key('renewal', now.add(1, 'month'), now.add(1, 'year'));
  const expired = key('expired', now.subtract(2, 'year'), now.subtract(1, 'year'));

  it('should prefer a valid key over one that is not valid yet over an expired one', () => {
    expect(toCustomerHomeLicense([expired, renewal, valid], [], now)?.name).toEqual('valid');
    expect(toCustomerHomeLicense([expired, renewal], [], now)?.name).toEqual('renewal');
    expect(toCustomerHomeLicense([expired], [], now)?.name).toEqual('expired');
  });

  it('should fall back to the entitlement that expires first, one without expiration last', () => {
    const entitlements = [
      {name: 'forever'},
      {name: 'late', expiresAt: now.add(1, 'year').toDate()},
      {name: 'early', expiresAt: now.add(1, 'month').toDate()},
    ];
    expect(toCustomerHomeLicense([], entitlements, now)?.name).toEqual('early');
    expect(toCustomerHomeLicense([], [], now)).toBeUndefined();
  });
});
