import {
  Application,
  ApplicationVersion,
  CustomerOrganization,
  DeploymentRevisionStatus,
  DeploymentStatusType,
  DeploymentTarget,
  DeploymentWithLatestRevision,
} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {LatestArtifactPull} from '../../types/artifact-version-pull';
import {License} from '../../types/license';
import {SupportBundle} from '../../types/support-bundle';
import {
  toVendorHomeCustomers,
  toVendorHomeDeploymentHealth,
  toVendorHomeLicenses,
  toVendorHomeUpdates,
} from './vendor-home-mappers';

const fresh = () => dayjs().toISOString();
const old = () => dayjs().subtract(1, 'hour').toISOString();

const status = (type: DeploymentStatusType, createdAt = fresh()): DeploymentRevisionStatus => ({
  createdAt,
  deploymentRevisionId: 'revision',
  type,
  message: '',
});

const deployment = (
  latestStatus?: DeploymentRevisionStatus,
  currentStatus?: DeploymentRevisionStatus
): DeploymentWithLatestRevision => ({
  deploymentTargetId: 'target',
  application: {name: 'app', type: 'docker'},
  applicationId: 'app',
  applicationName: 'app',
  applicationVersionId: 'version',
  applicationVersionName: '1.0.0',
  applicationLink: '',
  latestStatus,
  currentStatus,
});

const customer = (id: string): CustomerOrganization => ({
  id,
  createdAt: fresh(),
  name: id,
  features: [],
});

const target = (
  deployments: DeploymentWithLatestRevision[],
  customerOrganization?: CustomerOrganization
): DeploymentTarget => ({
  name: 'target',
  type: 'docker',
  customerOrganization,
  deployments,
  metricsEnabled: false,
  imageCleanupEnabled: false,
  deploymentLogsEnabled: false,
});

describe('toVendorHomeDeploymentHealth', () => {
  it('should count a stale deployment as stale only and leave out progressing and unknown ones', () => {
    const targets = [
      target([deployment(status('healthy')), deployment(status('running')), deployment(status('error'))]),
      target([
        deployment(status('healthy', old())),
        deployment(status('progressing')),
        deployment(),
        deployment(status('healthy')),
      ]),
    ];
    expect(toVendorHomeDeploymentHealth(targets)).toEqual({total: 7, healthy: 2, running: 1, error: 1, stale: 1});
  });

  it('should judge a deployment by its applied revision, not by a failed update', () => {
    const failedUpdate = deployment(status('error'), status('healthy'));
    expect(toVendorHomeDeploymentHealth([target([failedUpdate])])).toEqual({
      total: 1,
      healthy: 1,
      running: 0,
      error: 0,
      stale: 0,
    });
  });
});

const version = (id: string, createdAt: string): ApplicationVersion => ({id, name: id, createdAt});

const application: Application = {
  id: 'app',
  name: 'app',
  type: 'docker',
  versioningStrategy: 'semver',
  versions: [version('1.0.0', '2026-01-01T00:00:00Z'), version('2.0.0', '2026-02-01T00:00:00Z')],
};

const deploymentOf = (versionId: string): DeploymentWithLatestRevision => ({
  ...deployment(status('healthy')),
  application,
  applicationVersionId: versionId,
});

const pull = (customerOrganizationId: string, availableUpdate?: string): LatestArtifactPull => ({
  customerOrganizationId,
  artifactId: 'artifact',
  artifactName: 'artifact',
  versionName: '1.0.0',
  availableUpdate,
});

describe('toVendorHomeUpdates', () => {
  it('should judge a customer by its deployments and fall back to its artifact pulls without any', () => {
    const [current, outdated, pulledOutdated, pulledCurrent, idle] = ['a', 'b', 'c', 'd', 'e'].map(customer);
    const targets = [
      target([deploymentOf('2.0.0')], current),
      target([], current),
      target([deploymentOf('1.0.0'), deploymentOf('2.0.0')], outdated),
      target([deploymentOf('1.0.0')], outdated),
      target([], pulledOutdated),
      target([deploymentOf('1.0.0')]),
    ];
    const pulls = [pull('a', '2.0.0'), pull('c', '2.0.0'), pull('d')];
    expect(
      toVendorHomeUpdates([current, outdated, pulledOutdated, pulledCurrent, idle], targets, [application], [], pulls)
    ).toEqual({totalCustomers: 5, outdatedCustomers: 2, outdatedDeployments: 2});
  });
});

describe('toVendorHomeLicenses', () => {
  it('should count a customer with an expired license as expired only and ignore expirations after the window', () => {
    const now = dayjs('2026-06-01T00:00:00Z');
    const inDays = (days: number) => now.add(days, 'day').toISOString();
    const date = (expiresAt?: string) => (expiresAt === undefined ? undefined : new Date(expiresAt));
    const license = (
      id: string,
      {
        keys = [],
        applications = [],
        artifacts = [],
      }: Partial<Record<'keys' | 'applications' | 'artifacts', (string | undefined)[]>>
    ): License => ({
      customerOrganization: customer(id),
      licenseKeys: keys.map((expiresAt) => ({id: 'key', name: 'key', expiresAt})),
      applicationEntitlements: applications.map((expiresAt) => ({id: 'app', name: 'app', expiresAt: date(expiresAt)})),
      artifactEntitlements: artifacts.map((expiresAt) => ({
        id: 'artifact',
        name: 'artifact',
        expiresAt: date(expiresAt),
      })),
    });
    const licenses = [
      license('key', {keys: [inDays(10), inDays(20)]}),
      license('application', {applications: [inDays(30)]}),
      license('artifact', {artifacts: [inDays(400), inDays(1)]}),
      license('expired', {keys: [inDays(-1)], applications: [inDays(-10)]}),
      license('both', {keys: [inDays(5)], artifacts: [inDays(0)]}),
      license('later', {keys: [inDays(31)]}),
      license('unlimited', {keys: [undefined], applications: [undefined]}),
    ];
    expect(toVendorHomeLicenses(licenses, now)).toEqual({
      expiredCustomerIds: ['expired', 'both'],
      expiringCustomerIds: ['key', 'application', 'artifact'],
    });
  });
});

describe('toVendorHomeCustomers', () => {
  it('should count the open items of each customer and order customers by them, then by name', () => {
    const [zeta, alpha, errors, pulled, licensed, support] = [
      'zeta',
      'alpha',
      'errors',
      'pulled',
      'licensed',
      'support',
    ].map(customer);
    const targets = [
      target([deploymentOf('2.0.0')], zeta),
      target([{...deploymentOf('1.0.0'), latestStatus: status('error')}, deployment(status('healthy', old()))], errors),
      target([deployment(status('error'))]),
      target([], pulled),
    ];
    const bundle = (customerOrganizationId: string) => ({customerOrganizationId}) as SupportBundle;
    const result = toVendorHomeCustomers(
      [zeta, alpha, errors, pulled, licensed, support],
      targets,
      [application],
      [],
      [pull('pulled', '2.0.0'), pull('zeta', '2.0.0')],
      {expiredCustomerIds: ['licensed'], expiringCustomerIds: ['support']},
      [bundle('support'), bundle('support'), bundle('other')]
    );
    expect(result.map((c) => [c.id, c.openItems])).toEqual([
      ['errors', 3],
      ['support', 3],
      ['licensed', 1],
      ['pulled', 1],
      ['alpha', 0],
      ['zeta', 0],
    ]);
  });
});
