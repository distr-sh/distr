import {CustomerOrganizationWithUsage} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {License} from '../../types/license';
import {LicenseKey} from '../../types/license-key';
import {toCustomerLicenseStatus, toCustomerListRows} from './customer-list-mappers';

const now = dayjs('2026-06-01T00:00:00Z');
const inDays = (days: number) => now.add(days, 'day').toISOString();

const customer: CustomerOrganizationWithUsage = {
  id: 'customer',
  createdAt: inDays(-100),
  name: 'Acme Corp',
  features: [],
  userCount: 0,
  deploymentTargetCount: 0,
};

const license = ({
  keys = [],
  entitlements = [],
}: {
  keys?: Partial<LicenseKey>[];
  entitlements?: (string | undefined)[];
}): License => ({
  customerOrganization: customer,
  licenseKeys: keys.map((key) => ({name: 'key', ...key})),
  applicationEntitlements: entitlements.map((expiresAt) => ({
    name: 'entitlement',
    expiresAt: expiresAt === undefined ? undefined : new Date(expiresAt),
  })),
  artifactEntitlements: [],
});

describe('toCustomerLicenseStatus', () => {
  it('should report a customer without license keys and entitlements as having no license', () => {
    expect(toCustomerLicenseStatus(undefined, [], now)).toBe('no_license');
    expect(toCustomerLicenseStatus(license({}), [], now)).toBe('no_license');
  });

  it('should let an expired license win over an expiring one and count a license without expiration as valid', () => {
    expect(
      toCustomerLicenseStatus(license({keys: [{expiresAt: inDays(-1)}], entitlements: [inDays(1)]}), [], now)
    ).toBe('expired');
    expect(toCustomerLicenseStatus(license({keys: [{expiresAt: undefined}]}), [], now)).toBe('valid');
  });

  it('should count an expiration within 7 days as expiring', () => {
    expect(toCustomerLicenseStatus(license({entitlements: [inDays(5)]}), [], now)).toBe('expiring');
    expect(toCustomerLicenseStatus(license({entitlements: [inDays(10)]}), [], now)).toBe('valid');
  });

  it('should widen the window of a key to the grace period of its template', () => {
    const templates = [{id: 'template', name: 'template', payloadTemplate: '', expirationGracePeriodDays: 14}];
    const key = {expiresAt: inDays(10), licenseTemplateId: 'template'};
    expect(toCustomerLicenseStatus(license({keys: [key]}), templates, now)).toBe('expiring');
    expect(toCustomerLicenseStatus(license({keys: [{...key, licenseTemplateId: undefined}]}), templates, now)).toBe(
      'valid'
    );
  });
});

describe('toCustomerListRows', () => {
  it('should prefer controller management over self management', () => {
    const customers: CustomerOrganizationWithUsage[] = [
      {...customer, id: 'both', features: ['artifacts', 'deployment_targets']},
      {...customer, id: 'artifacts', features: ['artifacts']},
      {...customer, id: 'none', features: ['alerts']},
    ];
    expect(toCustomerListRows(customers, undefined, undefined, undefined, now).map((row) => row.management)).toEqual([
      'controller',
      'self',
      'none',
    ]);
  });
});
