import {AdvisoryApplicationVersion, VersioningStrategy} from '@distr-sh/distr-sdk';
import {sortAdvisoryApplicationVersions} from './versions';

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
});
