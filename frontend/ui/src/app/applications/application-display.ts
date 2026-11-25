import {VersioningStrategy} from '@distr-sh/distr-sdk';
import {selectableVersioningStrategies, versioningStrategyLabel} from '../../util/versions';
import {BadgeSelectOption} from '../components/badge-select/badge-select.component';

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
    variant: 'neutral',
  }));
}

export type AllowAutomaticUpdates = 'allowed' | 'disallowed';

export const allowAutomaticUpdatesSelectOptions: BadgeSelectOption<AllowAutomaticUpdates>[] = [
  {value: 'allowed', label: 'Auto-Updates allowed', variant: 'success'},
  {value: 'disallowed', label: 'Auto-Updates disallowed', variant: 'neutral'},
];
