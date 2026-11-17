import {CustomerOrganizationWithUsage, DeploymentTarget} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {LatestArtifactPull} from '../../types/artifact-version-pull';
import {License} from '../../types/license';
import {LicenseKey} from '../../types/license-key';
import {CustomerListDeploymentSources, toCustomerLicenseStatus, toCustomerListRows} from './customer-list-mappers';

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
    expect(toCustomerLicenseStatus(undefined, now)).toBe('no_license');
    expect(toCustomerLicenseStatus(license({}), now)).toBe('no_license');
  });

  it('should let an expired license win over an expiring one and count a license without expiration as valid', () => {
    expect(toCustomerLicenseStatus(license({keys: [{expiresAt: inDays(-1)}], entitlements: [inDays(1)]}), now)).toBe(
      'expired'
    );
    expect(toCustomerLicenseStatus(license({keys: [{expiresAt: undefined}]}), now)).toBe('valid');
  });

  it('should count an expiration within 30 days as expiring', () => {
    expect(toCustomerLicenseStatus(license({entitlements: [inDays(25)]}), now)).toBe('expiring');
    expect(toCustomerLicenseStatus(license({entitlements: [inDays(35)]}), now)).toBe('valid');
  });
});

describe('toCustomerListRows', () => {
  const managementOf = (customers: CustomerOrganizationWithUsage[], deployments?: CustomerListDeploymentSources) =>
    toCustomerListRows(customers, deployments, undefined, undefined, now).map((row) => row.management);

  it('should take the management from the features when the customer has only one of them', () => {
    const customers: CustomerOrganizationWithUsage[] = [
      {...customer, id: 'deployment_targets', features: ['deployment_targets']},
      {...customer, id: 'artifacts', features: ['artifacts']},
      {...customer, id: 'neither', features: ['alerts']},
    ];
    expect(managementOf(customers)).toEqual(['controller', 'self', 'unknown']);
  });

  it('should judge a customer with both features by its deployments first and its artifact pulls second', () => {
    const both = (id: string): CustomerOrganizationWithUsage => ({
      ...customer,
      id,
      features: ['artifacts', 'deployment_targets'],
    });
    const customers = [both('deployed'), both('deployed_and_pulled'), both('pulled'), both('idle')];
    const target = (customerId: string, deploymentCount: number): DeploymentTarget => ({
      name: 'target',
      type: 'docker',
      customerOrganization: customers.find((it) => it.id === customerId),
      deployments: Array.from({length: deploymentCount}, () => ({
        deploymentTargetId: 'target',
        application: {name: 'app', type: 'docker'},
        applicationId: 'app',
        applicationName: 'app',
        applicationVersionId: 'version',
        applicationVersionName: '1.0.0',
        applicationLink: '',
      })),
      metricsEnabled: false,
      imageCleanupEnabled: false,
      deploymentLogsEnabled: false,
    });
    const pull = (customerOrganizationId: string): LatestArtifactPull => ({
      customerOrganizationId,
      artifactId: 'artifact',
      artifactName: 'artifact',
      versionName: '1.0.0',
    });
    const deployments: CustomerListDeploymentSources = {
      targets: [target('deployed', 1), target('deployed_and_pulled', 1), target('idle', 0)],
      applications: [],
      entitlements: [],
      pulls: [pull('deployed_and_pulled'), pull('pulled')],
    };

    expect(managementOf(customers)).toEqual([undefined, undefined, undefined, undefined]);
    expect(managementOf(customers, deployments)).toEqual(['controller', 'controller', 'self', 'unknown']);
  });
});
