import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {CustomerOrganization, SidebarLink, UserAccountWithRole} from '@distr-sh/distr-sdk';
import posthog from 'posthog-js';
import {map, Observable, shareReplay, startWith, Subject, switchMap, tap} from 'rxjs';
import {Organization, OrganizationWithRole} from '../types/organization';

interface ContextResponse {
  user: UserAccountWithRole;
  organization: Organization;
  customerOrganization?: CustomerOrganization;
  sidebarLinks?: SidebarLink[];
  availableContexts?: OrganizationWithRole[];
}

/**
 * ContextService should not be used directly – use UsersService and OrganizationService instead to profit
 * from getting live updates as well.
 */
@Injectable({providedIn: 'root'})
export class ContextService {
  private readonly baseUrl = '/api/v1/context';
  private readonly httpClient = inject(HttpClient);
  private readonly reload$ = new Subject<void>();
  private readonly cache = this.reload$.pipe(
    startWith(undefined),
    switchMap(() => this.httpClient.get<ContextResponse>(this.baseUrl)),
    tap((ctx) => posthog.group('organization', ctx.organization.id!, {name: ctx.organization.name})),
    shareReplay(1)
  );

  public getUser(): Observable<UserAccountWithRole> {
    return this.cache.pipe(map((ctx) => ctx.user));
  }

  public getOrganization(): Observable<OrganizationWithRole> {
    return this.cache.pipe(
      map((ctx) => ({
        ...ctx.organization,
        accountRole: ctx.user.accountRole,
        userRole: ctx.user.accountRole,
        joinedOrgAt: ctx.user.joinedOrgAt,
      }))
    );
  }

  public getAvailableOrganizations(): Observable<OrganizationWithRole[]> {
    return this.cache.pipe(map((ctx) => ctx.availableContexts ?? []));
  }

  public getCustomerOrganization(): Observable<CustomerOrganization | undefined> {
    return this.cache.pipe(map((ctx) => ctx.customerOrganization));
  }

  public getSidebarLinks(): Observable<SidebarLink[]> {
    return this.cache.pipe(map((ctx) => ctx.sidebarLinks ?? []));
  }

  public reload() {
    this.reload$.next();
  }
}
