import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {CustomerOrganization, PartnerOrganization, SidebarLink, UserAccountWithRole} from '@distr-sh/distr-sdk';
import posthog from 'posthog-js';
import {map, Observable, shareReplay, startWith, Subject, switchMap, tap} from 'rxjs';
import {Organization, OrganizationWithUserRole} from '../types/organization';

interface ContextResponse {
  user: UserAccountWithRole;
  organization: Organization;
  customerOrganization?: CustomerOrganization;
  partnerOrganization?: PartnerOrganization;
  sidebarLinks?: SidebarLink[];
  availableContexts?: OrganizationWithUserRole[];
  registryHost?: string;
  canCreateOrganization: boolean;
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

  public getOrganization(): Observable<OrganizationWithUserRole> {
    return this.cache.pipe(
      map((ctx) => ({
        ...ctx.organization,
        userRole: ctx.user.userRole,
        customerOrganizationId: ctx.customerOrganization?.id,
        customerOrganizationName: ctx.customerOrganization?.name,
        partnerOrganizationId: ctx.partnerOrganization?.id,
        partnerOrganizationName: ctx.partnerOrganization?.name,
        joinedOrgAt: ctx.user.joinedOrgAt,
      }))
    );
  }

  public getAvailableOrganizations(): Observable<OrganizationWithUserRole[]> {
    return this.cache.pipe(map((ctx) => ctx.availableContexts ?? []));
  }

  public getCustomerOrganization(): Observable<CustomerOrganization | undefined> {
    return this.cache.pipe(map((ctx) => ctx.customerOrganization));
  }

  public getPartnerOrganization(): Observable<PartnerOrganization | undefined> {
    return this.cache.pipe(map((ctx) => ctx.partnerOrganization));
  }

  public getSidebarLinks(): Observable<SidebarLink[]> {
    return this.cache.pipe(map((ctx) => ctx.sidebarLinks ?? []));
  }

  /** The effective registry host of the organization, considering custom domains and legacy branding domains. */
  public getRegistryHost(): Observable<string | undefined> {
    return this.cache.pipe(map((ctx) => ctx.registryHost));
  }

  /**
   * Whether the user may create another organization. It is false for a user who signs in through an organization's
   * own identity provider: such an account has to stay a member of that one organization.
   */
  public canCreateOrganization(): Observable<boolean> {
    return this.cache.pipe(map((ctx) => ctx.canCreateOrganization));
  }

  public reload() {
    this.reload$.next();
  }
}
