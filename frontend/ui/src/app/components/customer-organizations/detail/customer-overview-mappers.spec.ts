import {Application, DeploymentTarget, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {LatestArtifactPull} from '../../../types/artifact-version-pull';
import {License} from '../../../types/license';
import {toCustomerOverviewLicense, toCustomerOverviewUpdates} from './customer-overview-mappers';

describe('toCustomerOverviewLicense', () => {
  const now = dayjs('2026-06-01T00:00:00Z');
  const inDays = (days: number) => now.add(days, 'day').toISOString();
  const license = (id: string, keys: (string | undefined)[], applications: (string | undefined)[] = []): License => ({
    customerOrganization: {id, createdAt: now.toISOString(), name: id, features: []},
    licenseKeys: keys.map((expiresAt) => ({id: 'key', name: 'key', expiresAt})),
    applicationEntitlements: applications.map((expiresAt) => ({
      id: 'app',
      name: 'app',
      expiresAt: expiresAt === undefined ? undefined : new Date(expiresAt),
    })),
    artifactEntitlements: [],
  });

  it('should count an expired grant as expired only and leave out those expiring after the window', () => {
    const licenses = [license('a', [inDays(-1), inDays(0), inDays(10)], [inDays(30), inDays(31), undefined])];
    expect(toCustomerOverviewLicense('a', licenses, now)).toEqual({
      status: 'expired',
      isLicensingInUse: true,
      grants: 6,
      expired: 2,
      expiring: 2,
    });
  });

  it('should consider licensing in use once another customer has a license', () => {
    expect(toCustomerOverviewLicense('a', [license('a', [])], now).isLicensingInUse).toBe(false);
    expect(toCustomerOverviewLicense('a', [license('b', [undefined])], now)).toEqual({
      status: 'no_license',
      isLicensingInUse: true,
      grants: 0,
      expired: 0,
      expiring: 0,
    });
  });
});

describe('toCustomerOverviewUpdates', () => {
  const application: Application = {
    id: 'app',
    name: 'app',
    type: 'docker',
    versioningStrategy: 'semver',
    versions: [
      {id: '1.0.0', name: '1.0.0', createdAt: '2026-01-01T00:00:00Z'},
      {id: '2.0.0', name: '2.0.0', createdAt: '2026-02-01T00:00:00Z'},
    ],
  };
  const deployment = (applicationVersionId: string): DeploymentWithLatestRevision => ({
    deploymentTargetId: 'target',
    application,
    applicationId: 'app',
    applicationName: 'app',
    applicationVersionId,
    applicationVersionName: applicationVersionId,
    applicationLink: '',
  });
  const target = (deployments: DeploymentWithLatestRevision[]): DeploymentTarget => ({
    name: 'target',
    type: 'docker',
    deployments,
    metricsEnabled: false,
    imageCleanupEnabled: false,
    deploymentLogsEnabled: false,
  });
  const pull = (availableUpdate?: string): LatestArtifactPull => ({
    customerOrganizationId: 'a',
    artifactId: 'artifact',
    artifactName: 'artifact',
    versionName: '1.0.0',
    availableUpdate,
  });

  it('should judge the customer by its deployments and by its pulls only without any', () => {
    const outdatedPulls = [pull('2.0.0'), pull(), pull('2.0.0')];
    expect(
      toCustomerOverviewUpdates(
        [target([deployment('1.0.0'), deployment('2.0.0')]), target([deployment('1.0.0')])],
        [application],
        [],
        outdatedPulls
      )
    ).toEqual({source: 'deployments', outdated: 2});
    expect(toCustomerOverviewUpdates([target([])], [application], [], outdatedPulls)).toEqual({
      source: 'downloads',
      outdated: 2,
    });
  });
});
