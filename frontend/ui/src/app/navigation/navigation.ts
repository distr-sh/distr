import {CustomerOrganizationFeature, SidebarLink} from '@distr-sh/distr-sdk';
import {IconDefinition} from '@fortawesome/angular-fontawesome';
import {FeatureLock, isLocked, upgradePlan} from '../types/feature-lock';
import {Feature} from '../types/organization';
import {UpgradePlan} from '../types/subscription';

export interface NavigationContext {
  user: {
    isAdmin: boolean;
    isAdminOrSuperAdmin: boolean;
    isReadWriteOrAdmin: boolean;
    isSuperAdmin: boolean;
  };
  organization: {
    hasSubscription: boolean;
    hasUnstartedTutorials: boolean;
  };
  features: ReadonlySet<Feature>;
  customer: {
    features: ReadonlySet<CustomerOrganizationFeature>;
    hasLicenseKeys: boolean;
    sidebarLinks: SidebarLink[];
  };
}

export type NavigationPredicate = (context: NavigationContext) => boolean;

interface NavigationEntryDefinitionBase {
  id: string;
  label: string;
  icon: IconDefinition;
  visible?: NavigationPredicate;
  /** Shows the entry with a plan badge and a plan hint instead of the page while the lock holds. */
  locked?: FeatureLock;
}

export interface NavigationLinkDefinition extends NavigationEntryDefinitionBase {
  route: string;
  indicator?: NavigationPredicate;
}

export interface NavigationExternalLinkDefinition extends NavigationEntryDefinitionBase {
  href: string;
}

export type NavigationEntryDefinition = NavigationLinkDefinition | NavigationExternalLinkDefinition;

/** A section of a portal's navigation. The sidebar renders a divider between sections and does not show the label. */
export interface NavigationSectionDefinition {
  id: string;
  label: string;
  entries: NavigationEntryDefinition[] | ((context: NavigationContext) => NavigationEntryDefinition[]);
}

export type NavigationDefinition = NavigationSectionDefinition[];

export type NavigationEntry =
  | {kind: 'link'; id: string; label: string; icon: IconDefinition; route: string; hasIndicator: boolean}
  | {kind: 'external'; id: string; label: string; icon: IconDefinition; href: string}
  | {kind: 'locked'; id: string; label: string; icon: IconDefinition; plan: UpgradePlan};

export interface NavigationSection {
  id: string;
  label: string;
  entries: NavigationEntry[];
}

/** Resolves the sections that have at least one visible entry. */
export function resolveNavigation(definition: NavigationDefinition, context: NavigationContext): NavigationSection[] {
  return definition
    .map(({id, label, entries}) => ({
      id,
      label,
      entries: (typeof entries === 'function' ? entries(context) : entries)
        .filter((entry) => entry.visible?.(context) ?? true)
        .map((entry) => resolveEntry(entry, context)),
    }))
    .filter((section) => section.entries.length > 0);
}

function resolveEntry(entry: NavigationEntryDefinition, context: NavigationContext): NavigationEntry {
  const {id, label, icon, locked} = entry;
  if (locked && isLocked(locked, context.features)) {
    return {kind: 'locked', id, label, icon, plan: upgradePlan(locked)};
  }
  if ('href' in entry) {
    return {kind: 'external', id, label, icon, href: entry.href};
  }
  return {kind: 'link', id, label, icon, route: entry.route, hasIndicator: entry.indicator?.(context) ?? false};
}
