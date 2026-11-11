import {inject} from '@angular/core';
import {CanActivateFn, RedirectFunction, Router} from '@angular/router';
import {map, Observable} from 'rxjs';
import {environment} from '../../env/env';
import {Edition} from '../../env/types';
import {TabItem} from '../components/tab-bar.component';
import {AuthService} from '../services/auth.service';
import {OrganizationService} from '../services/organization.service';
import {Feature} from '../types/organization';
import {UpgradePlan} from '../types/subscription';

export interface TabContext {
  edition: Edition;
  user: {isAdminOrSuperAdmin: boolean};
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
        user: {isAdminOrSuperAdmin: auth.isSuperAdmin() || auth.hasRole('admin')},
        features: new Set(org.features),
      }))
    );
}

export function defineRoutedTabs<T extends string>(
  path: string,
  pageTitle: string,
  definitions: TabDefinition<T>[]
): RoutedTabs<T> {
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
    guard: (id) => () => {
      const router = inject(Router);
      return openable().pipe(map((ids) => ids.includes(id) || router.createUrlTree([path])));
    },
    firstOpenable: () => openable().pipe(map(([id]) => (id ? `${path}/${id}` : '/'))),
  };
}
