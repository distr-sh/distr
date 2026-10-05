import {
  AdvisoryApplicationVersion,
  Application,
  ApplicationVersion,
  DeploymentWithLatestRevision,
  VersioningStrategy,
} from '@distr-sh/distr-sdk';
import {SemVer} from 'semver';
import {isArchived} from './dates';

export type VersionComparator = (a: ApplicationVersion, b: ApplicationVersion) => number;

/** The strategies an application may be set to. The legacy one only ever applies to existing ones. */
export type SelectableVersioningStrategy = Exclude<VersioningStrategy, 'legacy'>;

/**
 * Mirrors types.ApplicationVersionComparator in the backend, which decides which version an
 * automatic update moves a deployment to. The ordering is derived from the whole set because the
 * legacy strategy falls back to the creation date as soon as one name is not valid SemVer.
 */
export function applicationVersionComparator(
  strategy: VersioningStrategy | undefined,
  versions: ApplicationVersion[]
): VersionComparator {
  if (strategy === 'chronological') {
    return compareByCreationDate;
  }
  const parsed = new Map<string, SemVer>();
  for (const version of versions) {
    try {
      parsed.set(version.id!, new SemVer(version.name, {loose: true}));
    } catch {
      if (strategy !== 'semver') {
        return compareByCreationDate;
      }
    }
  }
  return (a, b) => {
    const parsedA = parsed.get(a.id!);
    const parsedB = parsed.get(b.id!);
    return parsedA && parsedB ? parsedA.compare(parsedB) : compareByCreationDate(a, b);
  };
}

function compareByCreationDate(a: ApplicationVersion, b: ApplicationVersion): number {
  return (a.createdAt ?? '').localeCompare(b.createdAt ?? '');
}

/**
 * Returns a new array ordered by the application's versioning strategy. Pass the full version set
 * as `allVersions` when sorting a subset so legacy/semver match the backend comparator. Defaults
 * to descending (newest / highest first) for UI lists.
 */
export function sortApplicationVersions(
  strategy: VersioningStrategy | undefined,
  versions: ApplicationVersion[],
  allVersions: ApplicationVersion[] = versions,
  direction: 'asc' | 'desc' = 'desc'
): ApplicationVersion[] {
  const compare = applicationVersionComparator(strategy, allVersions);
  return [...versions].sort((a, b) => (direction === 'desc' ? compare(b, a) : compare(a, b)));
}

export function sortAdvisoryApplicationVersions(versions: AdvisoryApplicationVersion[]): AdvisoryApplicationVersion[] {
  return [...versions].sort((a, b) => {
    const byApplication =
      a.applicationName.localeCompare(b.applicationName) || a.applicationId.localeCompare(b.applicationId);
    if (byApplication !== 0) {
      return byApplication;
    }
    const compare = applicationVersionComparator(
      a.applicationVersioningStrategy,
      versions.filter((v) => v.applicationId === a.applicationId).map(toApplicationVersion)
    );
    return compare(toApplicationVersion(b), toApplicationVersion(a));
  });
}

function toApplicationVersion(version: AdvisoryApplicationVersion): ApplicationVersion {
  return {
    id: version.applicationVersionId,
    name: version.applicationVersionName,
    createdAt: version.applicationVersionCreatedAt,
  };
}

/**
 * The newest version that is not archived, or undefined when none remains. The ordering comes from
 * applicationVersionComparator, so that a subset (the versions an entitlement covers) is ordered by
 * the ordering of the whole.
 */
export function latestApplicationVersion(
  compare: VersionComparator,
  versions: ApplicationVersion[]
): ApplicationVersion | undefined {
  return versions
    .filter((version) => !isArchived(version))
    .reduce<ApplicationVersion | undefined>(
      (latest, version) => (latest === undefined || compare(version, latest) > 0 ? version : latest),
      undefined
    );
}

export function allowsAutomaticUpdates(application: Application): boolean {
  return (application.allowAutomaticUpdates ?? false) && application.versioningStrategy !== 'legacy';
}

/**
 * The deployments whose application allows automatic updates. The deployment carries only a name
 * and an id of its application, so whether it allows them has to come from the application list.
 */
export function deploymentIdsAllowingAutomaticUpdates(
  deployments: DeploymentWithLatestRevision[],
  applications: Application[]
): Set<string> {
  return new Set(
    deployments
      .filter((deployment) => {
        const application = applications.find((app) => app.id === deployment.application.id);
        return application !== undefined && allowsAutomaticUpdates(application);
      })
      .map((deployment) => deployment.id)
      .filter((id) => id !== undefined)
  );
}

export function versioningStrategyLabel(strategy: VersioningStrategy | undefined): string {
  switch (strategy) {
    case 'semver':
      return 'Semantic versioning';
    case 'chronological':
      return 'Chronological versioning';
    default:
      return 'Legacy versioning';
  }
}

export function versioningStrategyBadgeClass(): string {
  return 'bg-gray-100 text-gray-800 border-gray-400 dark:bg-gray-600 dark:text-gray-200 dark:border-gray-500';
}

export function automaticUpdatesBadgeClass(enabled: boolean): string {
  return enabled
    ? 'bg-green-100 text-green-800 border-green-400 dark:bg-green-900 dark:text-green-300 dark:border-green-800'
    : 'bg-gray-100 text-gray-800 border-gray-400 dark:bg-gray-600 dark:text-gray-200 dark:border-gray-500';
}
