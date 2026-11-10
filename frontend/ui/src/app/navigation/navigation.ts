import {CustomerOrganizationFeature, SidebarLink} from '@distr-sh/distr-sdk';
import {IconDefinition} from '@fortawesome/angular-fontawesome';
import {Feature} from '../types/organization';
import {UpgradePlan} from '../types/subscription';

/** The organization features that show, hide or lock an entry, including those that follow from the plan alone. */
export type NavigationFeature =
  | Extract<Feature, 'licensing' | 'vulnerabilities' | 'vendor_billing' | 'partner_management'>
  | 'notifications'
  | 'support_bundles';

export interface NavigationContext {
  isEnterpriseEdition: boolean;
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
  features: ReadonlySet<NavigationFeature>;
  customer: {
    features: ReadonlySet<CustomerOrganizationFeature>;
    hasLicenseKeys: boolean;
    sidebarLinks: SidebarLink[];
  };
}

export type NavigationPredicate = (context: NavigationContext) => boolean;

export interface NavigationChildDefinition {
  id: string;
  label: string;
  route: string;
  visible?: NavigationPredicate;
}

interface NavigationEntryDefinitionBase {
  id: string;
  label: string;
  icon: IconDefinition;
  visible?: NavigationPredicate;
  /** Shows the entry with a plan badge and a plan hint instead of the page while the predicate holds. */
  locked?: {plan: UpgradePlan; when: NavigationPredicate};
}

export interface NavigationLinkDefinition extends NavigationEntryDefinitionBase {
  route: string;
  indicator?: NavigationPredicate;
}

export interface NavigationExternalLinkDefinition extends NavigationEntryDefinitionBase {
  href: string;
}

export interface NavigationGroupDefinition extends NavigationEntryDefinitionBase {
  children: NavigationChildDefinition[];
}

export type NavigationEntryDefinition =
  NavigationLinkDefinition | NavigationExternalLinkDefinition | NavigationGroupDefinition;

/** A section of a portal's navigation. The sidebar renders a divider between sections and does not show the label. */
export interface NavigationSectionDefinition {
  id: string;
  label: string;
  entries: NavigationEntryDefinition[] | ((context: NavigationContext) => NavigationEntryDefinition[]);
}

export type NavigationDefinition = NavigationSectionDefinition[];

export interface NavigationChild {
  id: string;
  label: string;
  route: string;
}

export type NavigationEntry =
  | {kind: 'link'; id: string; label: string; icon: IconDefinition; route: string; hasIndicator: boolean}
  | {kind: 'external'; id: string; label: string; icon: IconDefinition; href: string}
  | {kind: 'locked'; id: string; label: string; icon: IconDefinition; plan: UpgradePlan}
  | {kind: 'group'; id: string; label: string; icon: IconDefinition; children: NavigationChild[]};

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
  if (locked?.when(context)) {
    return {kind: 'locked', id, label, icon, plan: locked.plan};
  }
  if ('children' in entry) {
    const children = entry.children
      .filter((child) => child.visible?.(context) ?? true)
      .map(({id, label, route}) => ({id, label, route}));
    return {kind: 'group', id, label, icon, children};
  }
  if ('href' in entry) {
    return {kind: 'external', id, label, icon, href: entry.href};
  }
  return {kind: 'link', id, label, icon, route: entry.route, hasIndicator: entry.indicator?.(context) ?? false};
}
