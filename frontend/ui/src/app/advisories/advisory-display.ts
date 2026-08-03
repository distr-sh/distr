import {never} from '../../util/exhaust';
import {
  AdvisoryApplicationVersion,
  AdvisoryArtifactVersion,
  AdvisoryEventType,
  AdvisoryImpactState,
  AdvisorySeverity,
  AdvisoryStatus,
} from '../types/advisory';

/**
 * The full name of a marked version. The version panel truncates these to keep the sidebar
 * width stable, so the same string is also what goes into the title attribute.
 */
export function applicationVersionLabel(version: AdvisoryApplicationVersion): string {
  return `${version.applicationName} ${version.applicationVersionName}`;
}

export function artifactVersionLabel(version: AdvisoryArtifactVersion): string {
  const tags = version.artifactVersionTags.join(', ');
  return [version.artifactName, tags, version.artifactVersionDigest].filter((part) => part).join(' ');
}

export const advisorySeverities: AdvisorySeverity[] = ['none', 'low', 'medium', 'high', 'critical'];

export const advisoryStatuses: AdvisoryStatus[] = ['triage', 'draft', 'published', 'resolved', 'canceled'];

/**
 * The statuses the list shows before anyone touches the filter. Canceled advisories were
 * closed without disclosure and are rarely what you are looking for, so they stay out of the
 * way until explicitly asked for.
 */
export const defaultAdvisoryStatusFilter: AdvisoryStatus[] = advisoryStatuses.filter((status) => status !== 'canceled');

export function statusLabel(status: AdvisoryStatus): string {
  switch (status) {
    case 'triage':
      return 'Triage';
    case 'draft':
      return 'Draft';
    case 'published':
      return 'Active';
    case 'resolved':
      return 'Resolved';
    case 'canceled':
      return 'Canceled';
    default:
      return never(status);
  }
}

/**
 * What customers and partners see in place of the status. The editorial workflow is the
 * vendor's business; the only thing that matters on the other side is whether a deployment
 * still runs an affected version, which is what the `affected` flag reports.
 *
 * A missing flag means the caller is a vendor, who never reaches this.
 */
export function affectedLabel(affected: boolean | undefined): string {
  return affected ? 'Affected' : 'Not affected';
}

export function affectedBadgeClass(affected: boolean | undefined): string {
  return affected
    ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300'
    : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
}

export function severityLabel(severity: AdvisorySeverity): string {
  switch (severity) {
    case 'none':
      return 'None';
    case 'low':
      return 'Low';
    case 'medium':
      return 'Medium';
    case 'high':
      return 'High';
    case 'critical':
      return 'Critical';
    default:
      return never(severity);
  }
}

export function statusBadgeClass(status: AdvisoryStatus): string {
  switch (status) {
    case 'triage':
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
    case 'draft':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
    case 'published':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300';
    case 'resolved':
      return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
    case 'canceled':
      return 'bg-gray-200 text-gray-500 dark:bg-gray-700 dark:text-gray-400';
    default:
      return never(status);
  }
}

export function severityBadgeClass(severity: AdvisorySeverity): string {
  switch (severity) {
    case 'none':
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
    case 'low':
      return 'bg-sky-100 text-sky-800 dark:bg-sky-900 dark:text-sky-300';
    case 'medium':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300';
    case 'high':
      return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300';
    case 'critical':
      return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300';
    default:
      return never(severity);
  }
}

export function impactStateLabel(state: AdvisoryImpactState): string {
  switch (state) {
    case 'affected':
      return 'Affected';
    case 'fixed':
      return 'Fixed';
    case 'not_affected':
      return 'Not affected';
    default:
      return never(state);
  }
}

export function impactStateBadgeClass(state: AdvisoryImpactState): string {
  switch (state) {
    case 'affected':
      return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300';
    case 'fixed':
      return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
    case 'not_affected':
      return 'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200';
    default:
      return never(state);
  }
}

/** Mirrors advisoryStatusTransitions in internal/types/advisory.go. */
const statusTransitions: Record<AdvisoryStatus, AdvisoryStatus[]> = {
  triage: ['draft', 'canceled'],
  draft: ['triage', 'published', 'canceled'],
  published: ['draft', 'resolved'],
  resolved: ['published'],
  canceled: ['draft'],
};

export function allowedStatusTransitions(status: AdvisoryStatus): AdvisoryStatus[] {
  return statusTransitions[status];
}

/**
 * The transitions offered directly in the list, which are the ones that move an advisory
 * along its expected path. The rarer corrections, such as unpublishing or pushing a draft
 * back into triage, stay on the detail page where there is room to explain them.
 */
const quickStatusTransitions: Record<AdvisoryStatus, AdvisoryStatus[]> = {
  triage: ['draft', 'canceled'],
  draft: ['published', 'canceled'],
  published: ['resolved'],
  resolved: [],
  canceled: ['draft'],
};

export function quickStatusTransitionsFor(status: AdvisoryStatus): AdvisoryStatus[] {
  return quickStatusTransitions[status];
}

export function statusActionLabel(target: AdvisoryStatus): string {
  switch (target) {
    case 'triage':
      return 'Move back to triage';
    case 'draft':
      return 'Move to draft';
    case 'published':
      return 'Publish';
    case 'resolved':
      return 'Mark as resolved';
    case 'canceled':
      return 'Cancel';
    default:
      return never(target);
  }
}

/**
 * Compact labels for the list, where the button sits in a table cell next to the status
 * badge that already says where the advisory is coming from.
 */
export function statusActionShortLabel(from: AdvisoryStatus, target: AdvisoryStatus): string {
  if (from === 'canceled' && target === 'draft') {
    return 'Reopen';
  }
  switch (target) {
    case 'triage':
      return 'To triage';
    case 'draft':
      return 'To draft';
    case 'published':
      return 'Publish';
    case 'resolved':
      return 'Resolve';
    case 'canceled':
      return 'Cancel';
    default:
      return never(target);
  }
}

/**
 * The confirmation to show before a status change, or undefined when it needs none. Both the
 * list and the detail page ask before a step that is visible to customers or that ends the
 * advisory's life, so the wording lives here rather than in either component.
 */
export function statusChangeConfirmation(target: AdvisoryStatus): string | undefined {
  switch (target) {
    case 'published':
      return 'Publishing makes this advisory visible to customers who deployed or are entitled to an affected version. Continue?';
    case 'canceled':
      return 'Canceling closes this advisory without disclosing it. You can reopen it into draft later. Continue?';
    default:
      return undefined;
  }
}

export function eventLabel(type: AdvisoryEventType): string {
  switch (type) {
    case 'created':
      return 'created this advisory';
    case 'status_changed':
      return 'changed the status';
    case 'edited':
      return 'edited the details';
    case 'tags_changed':
      return 'changed the tags';
    case 'versions_changed':
      return 'changed the affected versions';
    case 'reference_added':
      return 'added a reference';
    case 'reference_removed':
      return 'removed a reference';
    case 'comment':
      return 'commented';
    default:
      return never(type);
  }
}
