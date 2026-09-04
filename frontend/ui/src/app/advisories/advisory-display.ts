import {
  AdvisoryApplicationVersion,
  AdvisoryArtifactVersion,
  AdvisoryEventType,
  AdvisoryImpactState,
  AdvisorySeverity,
  AdvisoryStatus,
} from '@distr-sh/distr-sdk';
import {firstValueFrom} from 'rxjs';
import {never} from '../../util/exhaust';
import {BadgeSelectOption} from '../components/badge-select/badge-select.component';
import {ConfirmConfig} from '../components/confirm-dialog/confirm-dialog.component';
import {OverlayService} from '../services/overlay.service';

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
 * Canceled advisories were closed without disclosure, so they stay out of the way until a
 * vendor asks for them explicitly.
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
 * What customers and partners see in place of the status. A missing flag means the caller is a
 * vendor, who never reaches this.
 */
export function affectedLabel(affected: boolean | undefined): string {
  return affected ? 'Affected' : 'Not affected';
}

export function affectedBadgeClass(affected: boolean | undefined): string {
  return affected
    ? 'bg-red-100 text-red-800 border-red-400 dark:bg-red-900 dark:text-red-300 dark:border-red-800'
    : 'bg-green-100 text-green-800 border-green-400 dark:bg-green-900 dark:text-green-300 dark:border-green-800';
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
      return 'bg-gray-100 text-gray-800 border-gray-400 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-600';
    case 'draft':
      return 'bg-blue-100 text-blue-800 border-blue-400 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800';
    case 'published':
      return 'bg-yellow-100 text-yellow-800 border-yellow-400 dark:bg-yellow-900 dark:text-yellow-300 dark:border-yellow-800';
    case 'resolved':
      return 'bg-green-100 text-green-800 border-green-400 dark:bg-green-900 dark:text-green-300 dark:border-green-800';
    case 'canceled':
      return 'bg-gray-200 text-gray-500 border-gray-400 dark:bg-gray-700 dark:text-gray-400 dark:border-gray-600';
    default:
      return never(status);
  }
}

export function severityBadgeClass(severity: AdvisorySeverity): string {
  switch (severity) {
    case 'none':
      return 'bg-gray-100 text-gray-800 border-gray-400 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-600';
    case 'low':
      return 'bg-sky-100 text-sky-800 border-sky-400 dark:bg-sky-900 dark:text-sky-300 dark:border-sky-800';
    case 'medium':
      return 'bg-yellow-100 text-yellow-800 border-yellow-400 dark:bg-yellow-900 dark:text-yellow-300 dark:border-yellow-800';
    case 'high':
      return 'bg-orange-100 text-orange-800 border-orange-400 dark:bg-orange-900 dark:text-orange-300 dark:border-orange-800';
    case 'critical':
      return 'bg-red-100 text-red-800 border-red-400 dark:bg-red-900 dark:text-red-300 dark:border-red-800';
    default:
      return never(severity);
  }
}

export function impactStateLabel(state: AdvisoryImpactState): string {
  switch (state) {
    case 'affected':
      return 'Affected';
    case 'patched':
      return 'Patched';
    case 'not_affected':
      return 'Not affected';
    default:
      return never(state);
  }
}

export function impactStateBadgeClass(state: AdvisoryImpactState): string {
  switch (state) {
    case 'affected':
      return 'bg-red-100 text-red-800 border-red-400 dark:bg-red-900 dark:text-red-300 dark:border-red-800';
    case 'patched':
      return 'bg-green-100 text-green-800 border-green-400 dark:bg-green-900 dark:text-green-300 dark:border-green-800';
    case 'not_affected':
      return 'bg-gray-100 text-gray-800 border-gray-400 dark:bg-gray-600 dark:text-gray-200 dark:border-gray-500';
    default:
      return never(state);
  }
}

export const statusSelectOptions: BadgeSelectOption<AdvisoryStatus>[] = advisoryStatuses.map((status) => ({
  value: status,
  label: statusLabel(status),
  badgeClass: statusBadgeClass(status),
}));

export const severitySelectOptions: BadgeSelectOption<AdvisorySeverity>[] = advisorySeverities.map((severity) => ({
  value: severity,
  label: severityLabel(severity),
  badgeClass: severityBadgeClass(severity),
}));

function isCustomerVisibleStatus(status: AdvisoryStatus): boolean {
  return status === 'published' || status === 'resolved';
}

export async function confirmAdvisoryVisibilityChange(
  overlay: OverlayService,
  from: AdvisoryStatus,
  to: AdvisoryStatus
): Promise<boolean> {
  if (isCustomerVisibleStatus(from) === isCustomerVisibleStatus(to)) {
    return true;
  }
  const config: ConfirmConfig = isCustomerVisibleStatus(to)
    ? {
        message: {
          message: `Set this advisory to ${statusLabel(to)}?`,
          alert: {
            type: 'warning',
            message:
              'Customers who deployed or are entitled to an affected version will see this advisory, ' +
              'and so will their partners.',
          },
        },
        confirmLabel: 'Disclose advisory',
      }
    : {
        message: {
          message: `Set this advisory to ${statusLabel(to)}?`,
          alert: {
            type: 'warning',
            message: 'Customers and partners will no longer see this advisory.',
          },
        },
        confirmLabel: 'Withdraw advisory',
      };
  return (await firstValueFrom(overlay.confirm(config))) ?? false;
}

export function eventLabel(type: AdvisoryEventType): string {
  switch (type) {
    case 'published':
      return 'published this advisory';
    case 'status_changed':
      return 'changed the status';
    case 'edited':
      return 'edited this advisory';
    case 'comment':
      return 'commented';
    default:
      return never(type);
  }
}
