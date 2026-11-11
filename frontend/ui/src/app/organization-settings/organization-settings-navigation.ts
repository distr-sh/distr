import {inject} from '@angular/core';
import {CanActivateFn, RedirectFunction, ResolveFn, Router} from '@angular/router';
import {map, Observable} from 'rxjs';
import {environment} from '../../env/env';
import {Edition} from '../../env/types';
import {TabItem} from '../components/tab-bar.component';
import {AuthService} from '../services/auth.service';
import {OrganizationService} from '../services/organization.service';
import {Feature} from '../types/organization';
import {UpgradePlan} from '../types/subscription';

export type SettingsTabId = 'general' | 'identity-provider' | 'email' | 'subscription';

export interface SettingsTabContext {
  edition: Edition;
  user: {
    isAdminOrSuperAdmin: boolean;
  };
  features: ReadonlySet<Feature>;
}

type SettingsTabPredicate = (context: SettingsTabContext) => boolean;

interface SettingsTabDefinition {
  id: SettingsTabId;
  label: string | ((context: SettingsTabContext) => string);
  visible?: SettingsTabPredicate;
  /** Shows the tab with a plan badge and a plan hint instead of the page while the predicate holds. */
  locked?: {plan: UpgradePlan; when: SettingsTabPredicate};
}

export interface SettingsTab extends TabItem<SettingsTabId> {
  plan?: UpgradePlan;
}

const settingsTabDefinitions: SettingsTabDefinition[] = [
  {id: 'general', label: 'General', visible: (c) => c.user.isAdminOrSuperAdmin},
  {
    id: 'identity-provider',
    // Drops the identity provider mention once domains are enabled without it, so the tab does not
    // advertise a capability the organization does not have.
    label: (c) =>
      c.features.has('custom_domains') && !c.features.has('custom_oidc_providers')
        ? 'Custom Domains'
        : 'Custom Domains & Identity Provider',
    visible: (c) => c.user.isAdminOrSuperAdmin,
    // The tab holds the custom domains as well, which are gated on their own feature. Both are
    // granted by the same plan, so either one is enough to make the tab worth opening.
    locked: {
      plan: 'business',
      when: (c) => !c.features.has('custom_oidc_providers') && !c.features.has('custom_domains'),
    },
  },
  {
    id: 'email',
    label: 'Email Sending Provider',
    visible: (c) => c.user.isAdminOrSuperAdmin,
    locked: {plan: 'business', when: (c) => !c.features.has('custom_emails')},
  },
  {
    id: 'subscription',
    label: 'Subscription',
    visible: (c) => c.user.isAdminOrSuperAdmin && c.edition !== 'community',
  },
];

export const settingsTabIds = settingsTabDefinitions.map((tab) => tab.id);

function resolveLabel({label}: SettingsTabDefinition, context: SettingsTabContext): string {
  return typeof label === 'function' ? label(context) : label;
}

export function resolveSettingsTabs(context: SettingsTabContext): SettingsTab[] {
  return settingsTabDefinitions
    .filter((tab) => tab.visible?.(context) ?? true)
    .map((tab) => {
      const label = resolveLabel(tab, context);
      return tab.locked?.when(context)
        ? {id: tab.id, label, disabled: true, plan: tab.locked.plan}
        : {id: tab.id, label};
    });
}

export function settingsTabContext(): Observable<SettingsTabContext> {
  const auth = inject(AuthService);
  return inject(OrganizationService)
    .get()
    .pipe(
      map((org) => ({
        edition: environment.edition,
        user: {isAdminOrSuperAdmin: auth.isSuperAdmin() || auth.hasRole('admin')},
        features: new Set(org.features),
      }))
    );
}

function openableSettingsTabs(): Observable<SettingsTabId[]> {
  return settingsTabContext().pipe(
    map((context) =>
      resolveSettingsTabs(context)
        .filter((tab) => !tab.disabled)
        .map((tab) => tab.id)
    )
  );
}

export function settingsTabTitle(id: SettingsTabId): ResolveFn<string> {
  const tab = settingsTabDefinitions.find((tab) => tab.id === id)!;
  return () => settingsTabContext().pipe(map((context) => `${resolveLabel(tab, context)} | Organization Settings`));
}

export function canOpenSettingsTab(tab: SettingsTabId): Observable<boolean> {
  return openableSettingsTabs().pipe(map((tabs) => tabs.includes(tab)));
}

export function settingsTabGuard(tab: SettingsTabId): CanActivateFn {
  return () => {
    const router = inject(Router);
    return openableSettingsTabs().pipe(
      map((tabs) => tabs.includes(tab) || router.createUrlTree(['/settings/organization']))
    );
  };
}

export const firstOpenableSettingsTab: RedirectFunction = () =>
  openableSettingsTabs().pipe(map(([tab]) => (tab ? `/settings/organization/${tab}` : '/')));
