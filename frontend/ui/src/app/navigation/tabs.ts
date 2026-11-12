import {inject} from '@angular/core';
import {CanActivateFn, createUrlTreeFromSnapshot, RedirectFunction} from '@angular/router';
import {map, Observable} from 'rxjs';
import {environment} from '../../env/env';
import {Edition} from '../../env/types';
import {TabItem} from '../components/tab-bar.component';
import {AuthService} from '../services/auth.service';
import {OrganizationService} from '../services/organization.service';
import {Feature} from '../types/organization';
import {isProSubscription, UpgradePlan} from '../types/subscription';

export interface TabContext {
  edition: Edition;
  user: {isAdminOrSuperAdmin: boolean; isPartner: boolean};
  organization: {isPro: boolean};
  features: ReadonlySet<Feature>;
}

type TabPredicate = (context: TabContext) => boolean;

export interface TabDefinition<T extends string> {
  id: T;
  label: string;
  visible?: TabPredicate;
  locked?: {plan: UpgradePlan; when: TabPredicate};
}

export interface LockableTab<T extends string = string> extends TabItem<T> {
  plan?: UpgradePlan;
}

export interface RoutedTabs<T extends string> {
  readonly ids: readonly T[];
  resolve(context: TabContext): LockableTab<T>[];
  title(id: T): string;
  canOpen(id: T): Observable<boolean>;
  guard(id: T): CanActivateFn;
  readonly firstOpenable: RedirectFunction;
}

export function tabContext(): Observable<TabContext> {
  const auth = inject(AuthService);
  return inject(OrganizationService)
    .get()
    .pipe(
      map((org) => ({
        edition: environment.edition,
        user: {isAdminOrSuperAdmin: auth.isSuperAdmin() || auth.hasRole('admin'), isPartner: auth.isPartner()},
        organization: {isPro: isProSubscription(org.subscriptionType)},
        features: new Set(org.features),
      }))
    );
}

/**
 * Defines the tabs of a routed page. The tabs have to be the direct children of the page's route, since the guard
 * sends a user who may not open a tab to the page and the redirect to the first openable tab is relative.
 */
export function defineRoutedTabs<T extends string>(pageTitle: string, definitions: TabDefinition<T>[]): RoutedTabs<T> {
  const resolve = (context: TabContext): LockableTab<T>[] =>
    definitions
      .filter((tab) => tab.visible?.(context) ?? true)
      .map(({id, label, locked}) =>
        locked?.when(context) ? {id, label, disabled: true, plan: locked.plan} : {id, label}
      );
  const openable = (): Observable<T[]> =>
    tabContext().pipe(
      map((context) =>
        resolve(context)
          .filter((tab) => !tab.disabled)
          .map((tab) => tab.id)
      )
    );
  return {
    ids: definitions.map((tab) => tab.id),
    resolve,
    title: (id) => `${definitions.find((tab) => tab.id === id)!.label} | ${pageTitle}`,
    canOpen: (id) => openable().pipe(map((ids) => ids.includes(id))),
    guard: (id) => (route) =>
      openable().pipe(map((ids) => ids.includes(id) || createUrlTreeFromSnapshot(route, ['..']))),
    firstOpenable: () => openable().pipe(map(([id]) => id ?? '/')),
  };
}
