import {VersioningStrategy} from '@distr-sh/distr-sdk';
import {
  automaticUpdatesBadgeClass,
  SelectableVersioningStrategy,
  versioningStrategyBadgeClass,
  versioningStrategyLabel,
} from '../../util/versions';
import {BadgeSelectOption} from '../components/badge-select/badge-select.component';

const selectableVersioningStrategies: SelectableVersioningStrategy[] = ['semver', 'chronological'];

/**
 * Legacy is listed only while the application is on it, since an application cannot be moved back
 * to it once it has a strategy of its own.
 */
export function versioningStrategySelectOptions(
  current: VersioningStrategy | undefined
): BadgeSelectOption<VersioningStrategy>[] {
  const strategies: VersioningStrategy[] =
    current === 'semver' || current === 'chronological'
      ? selectableVersioningStrategies
      : ['legacy', ...selectableVersioningStrategies];
  return strategies.map((strategy) => ({
    value: strategy,
    label: versioningStrategyLabel(strategy),
    badgeClass: versioningStrategyBadgeClass(),
  }));
}

export type AllowAutomaticUpdates = 'allowed' | 'disallowed';

export const allowAutomaticUpdatesSelectOptions: BadgeSelectOption<AllowAutomaticUpdates>[] = [
  {value: 'allowed', label: 'Auto-Updates allowed', badgeClass: automaticUpdatesBadgeClass(true)},
  {value: 'disallowed', label: 'Auto-Updates disallowed', badgeClass: automaticUpdatesBadgeClass(false)},
];
