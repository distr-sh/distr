import {
  AdvisoryApplicationVersion,
  Application,
  ApplicationVersion,
  DeploymentWithLatestRevision,
  VersioningStrategy,
} from '@distr-sh/distr-sdk';
import {ApplicationEntitlement} from '../app/types/application-entitlement';
import {isDeploymentOutdated, sortAdvisoryApplicationVersions} from './versions';

let nextVersionId = 0;

function advisoryVersion(
  applicationId: string,
  applicationName: string,
  applicationVersioningStrategy: VersioningStrategy,
  name: string,
  createdAt: string
): AdvisoryApplicationVersion {
  return {
    applicationId,
    applicationName,
    applicationType: 'docker',
    applicationVersioningStrategy,
    applicationVersion: {id: `version-${nextVersionId++}`, name, createdAt},
    relation: 'affected',
  };
}

function advisoryNames(versions: AdvisoryApplicationVersion[]): string[] {
  return versions.map((version) => `${version.applicationName} ${version.applicationVersion.name}`);
}

describe('sortAdvisoryApplicationVersions', () => {
  const base = '2026-01-01T00:00:00Z';
  const later = '2026-01-01T00:01:00Z';

  it('orders application names that are prefixes of each other', () => {
    // "App" gets the highest id, so its name and id concatenated sort after "App 2" and "AppB".
    const app = 'ffffffff-ffff-ffff-ffff-ffffffffffff';
    const appTwo = '00000000-0000-0000-0000-000000000001';
    const appB = '00000000-0000-0000-0000-000000000002';
    const versions = [
      advisoryVersion(appB, 'AppB', 'semver', '1.0.0', base),
      advisoryVersion(appTwo, 'App 2', 'semver', '1.0.0', base),
      advisoryVersion(app, 'App', 'semver', '1.0.0', base),
    ];

    expect(advisoryNames(sortAdvisoryApplicationVersions(versions))).toEqual([
      'App 1.0.0',
      'App 2 1.0.0',
      'AppB 1.0.0',
    ]);
  });

  it('keeps applications with the same name apart', () => {
    const first = '11111111-1111-1111-1111-111111111111';
    const second = '22222222-2222-2222-2222-222222222222';
    const versions = [
      advisoryVersion(second, 'App', 'chronological', '2.0.0', base),
      advisoryVersion(first, 'App', 'semver', '1.10.0', base),
      advisoryVersion(second, 'App', 'chronological', '1.0.0', later),
      advisoryVersion(first, 'App', 'semver', '1.9.0', later),
    ];

    expect(advisoryNames(sortAdvisoryApplicationVersions(versions))).toEqual([
      'App 1.10.0',
      'App 1.9.0',
      'App 1.0.0',
      'App 2.0.0',
    ]);
  });

  it('falls back to the creation date for a legacy name that only loose SemVer accepts', () => {
    const app = '11111111-1111-1111-1111-111111111111';
    const versions = [
      advisoryVersion(app, 'App', 'legacy', '2.0.0', base),
      advisoryVersion(app, 'App', 'legacy', '01.0.0', later),
    ];

    expect(advisoryNames(sortAdvisoryApplicationVersions(versions))).toEqual(['App 01.0.0', 'App 2.0.0']);
  });
});

describe('isDeploymentOutdated', () => {
  const version = (id: string, createdAt: string, archivedAt?: string): ApplicationVersion => ({
    id,
    name: id,
    createdAt,
    archivedAt,
  });

  const application: Application = {
    id: 'app',
    name: 'app',
    type: 'docker',
    versioningStrategy: 'semver',
    versions: [
      version('1.0.0', '2026-01-01T00:00:00Z'),
      version('1.1.0', '2026-03-01T00:00:00Z'),
      // Released after 2.0.0, which semver has to rank below it nonetheless.
      version('1.2.0', '2026-05-01T00:00:00Z'),
      version('2.0.0', '2026-04-01T00:00:00Z'),
      version('3.0.0', '2026-06-01T00:00:00Z', '2026-07-01T00:00:00Z'),
    ],
  };

  const deployment = (
    versionId: string,
    options: {currentVersionId?: string; entitlementId?: string} = {}
  ): DeploymentWithLatestRevision => ({
    deploymentTargetId: 'target',
    application: {id: 'app', name: 'app', type: 'docker'},
    applicationId: 'app',
    applicationName: 'app',
    applicationVersionId: versionId,
    applicationVersionName: versionId,
    applicationLink: '',
    currentApplicationVersionId: options.currentVersionId,
    applicationEntitlementId: options.entitlementId,
  });

  const entitlement = (id: string, versionIds: string[], expiresAt?: Date): ApplicationEntitlement => ({
    id,
    name: id,
    expiresAt,
    versions: application.versions!.filter((it) => versionIds.includes(it.id!)),
  });

  const outdated = (
    d: DeploymentWithLatestRevision,
    entitlements: ApplicationEntitlement[] = [],
    applications = [application]
  ) => isDeploymentOutdated(d, applications, entitlements);

  it('should compare by the versioning strategy and ignore archived versions', () => {
    expect(outdated(deployment('1.2.0'))).toBe(true);
    expect(outdated(deployment('2.0.0'))).toBe(false);
  });

  it('should judge the version the deployment runs rather than the one being rolled out', () => {
    expect(outdated(deployment('2.0.0', {currentVersionId: '1.0.0'}))).toBe(true);
    expect(outdated(deployment('1.0.0', {currentVersionId: '2.0.0'}))).toBe(false);
  });

  it('should compare against the newest version of the entitlement only', () => {
    const entitlements = [entitlement('limited', ['1.0.0', '1.1.0']), entitlement('expired', ['2.0.0'], new Date(0))];
    expect(outdated(deployment('1.0.0', {entitlementId: 'limited'}), entitlements)).toBe(true);
    expect(outdated(deployment('1.1.0', {entitlementId: 'limited'}), entitlements)).toBe(false);
    expect(outdated(deployment('2.0.0', {entitlementId: 'limited'}), entitlements)).toBe(false);
    expect(outdated(deployment('1.0.0', {entitlementId: 'expired'}), entitlements)).toBe(false);
  });

  it('should count a version the caller does not see as outdated unless it is the newest', () => {
    const customerView = {...application, versions: application.versions!.filter((it) => it.id !== '1.0.0')};
    expect(outdated(deployment('1.0.0'), [], [customerView])).toBe(true);
  });
});
