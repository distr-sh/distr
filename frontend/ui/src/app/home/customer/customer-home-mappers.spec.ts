import dayjs from 'dayjs';
import {LicenseKey} from '../../types/license-key';
import {toCustomerHomeLicense} from './customer-home-mappers';

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

  it('should count a key without expiration as valid', () => {
    expect(toCustomerHomeLicense([expired, {name: 'perpetual'}], [], now)?.name).toEqual('perpetual');
  });

  it('should let an entitlement that expires before the key decide despite a valid key', () => {
    const expiringEntitlement = {name: 'expiring', expiresAt: now.add(1, 'week').toDate()};
    const expiredEntitlement = {name: 'expired entitlement', expiresAt: now.subtract(1, 'day').toDate()};
    const laterEntitlement = {name: 'later', expiresAt: now.add(1, 'year').toDate()};
    expect(toCustomerHomeLicense([valid], [laterEntitlement, expiringEntitlement], now)?.name).toEqual('expiring');
    expect(toCustomerHomeLicense([valid], [expiredEntitlement], now)?.name).toEqual('expired entitlement');
    expect(toCustomerHomeLicense([valid], [laterEntitlement], now)?.name).toEqual('valid');
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
