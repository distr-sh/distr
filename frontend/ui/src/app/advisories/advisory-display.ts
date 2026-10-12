import {AdvisoryEventType, AdvisoryImpactState, AdvisorySeverity, AdvisoryStatus} from '@distr-sh/distr-sdk';
import {firstValueFrom} from 'rxjs';
import {never} from '../../util/exhaust';
import {BadgeVariant} from '../components/badge';
import {BadgeSelectOption} from '../components/badge-select/badge-select.component';
import {ConfirmConfig} from '../components/confirm-dialog/confirm-dialog.component';
import {OverlayService} from '../services/overlay.service';

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

export function affectedVariant(affected: boolean | undefined): BadgeVariant {
  return affected ? 'error' : 'success';
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

export function statusVariant(status: AdvisoryStatus): BadgeVariant {
  switch (status) {
    case 'triage':
      return 'neutral';
    case 'draft':
      return 'info';
    case 'published':
      return 'warning';
    case 'resolved':
      return 'success';
    case 'canceled':
      return 'muted';
    default:
      return never(status);
  }
}

export function severityVariant(severity: AdvisorySeverity): BadgeVariant {
  switch (severity) {
    case 'none':
      return 'neutral';
    case 'low':
      return 'info';
    case 'medium':
      return 'warning';
    case 'high':
      return 'severe';
    case 'critical':
      return 'error';
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

export function impactStateVariant(state: AdvisoryImpactState): BadgeVariant {
  switch (state) {
    case 'affected':
      return 'error';
    case 'patched':
      return 'success';
    case 'not_affected':
      return 'neutral';
    default:
      return never(state);
  }
}

export const statusSelectOptions: BadgeSelectOption<AdvisoryStatus>[] = advisoryStatuses.map((status) => ({
  value: status,
  label: statusLabel(status),
  variant: statusVariant(status),
}));

export const severitySelectOptions: BadgeSelectOption<AdvisorySeverity>[] = advisorySeverities.map((severity) => ({
  value: severity,
  label: severityLabel(severity),
  variant: severityVariant(severity),
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
