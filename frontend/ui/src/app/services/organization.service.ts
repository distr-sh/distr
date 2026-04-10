import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {combineLatestWith, map, merge, Observable, shareReplay, Subject, tap} from 'rxjs';
import {CreateUpdateOrganizationRequest, Organization, OrganizationWithUserRole} from '../types/organization';
import {ContextService} from './context.service';

@Injectable({
  providedIn: 'root',
})
export class OrganizationService {
  private readonly httpClient = inject(HttpClient);
  private readonly contextService = inject(ContextService);
  private readonly baseUrl = '/api/v1/organization';

  private readonly organizationUpdate = new Subject<OrganizationWithUserRole>();
  private readonly organization$ = merge(
    this.organizationUpdate.asObservable(),
    this.contextService.getOrganization()
  ).pipe(shareReplay(1));

  public readonly hasNoSubscription = toSignal(
    this.organization$.pipe(
      map(
        (org) =>
          !(
            org.subscriptionType === 'starter' ||
            org.subscriptionType === 'pro' ||
            org.subscriptionType === 'enterprise'
          )
      )
    ),
    {initialValue: false}
  );

  get(): Observable<OrganizationWithUserRole> {
    return this.organization$.pipe(shareReplay(1));
  }

  getAll(): Observable<OrganizationWithUserRole[]> {
    // TODO take updates into account like with organization$
    return this.contextService.getAvailableOrganizations();
  }

  create(name: string): Observable<Organization> {
    return this.httpClient.post<Organization>(this.baseUrl, {name});
  }

  update(organization: CreateUpdateOrganizationRequest): Observable<Organization> {
    return this.httpClient.put<Organization>(this.baseUrl, organization).pipe(
      combineLatestWith(this.getAll()),
      map(([it, allOrgs]) => {
        const foundOrg = allOrgs.find((o) => o.id === it.id);
        return {
          ...it,
          userRole: foundOrg?.userRole!,
          joinedOrgAt: foundOrg?.joinedOrgAt ?? new Date().toISOString(),
        };
      }),
      tap((it: OrganizationWithUserRole) => {
        this.organizationUpdate.next(it);
      })
    );
  }

  delete(): Observable<void> {
    return this.httpClient.delete<void>(this.baseUrl);
  }

  updateWebhookSecret(webhookSecret: string): Observable<void> {
    return this.httpClient
      .put<void>(`${this.baseUrl}/webhook`, {webhookSecret})
      .pipe(tap(() => this.contextService.reload()));
  }

  deleteWebhookSecret(): Observable<void> {
    return this.httpClient
      .put<void>(`${this.baseUrl}/webhook`, {webhookSecret: null})
      .pipe(tap(() => this.contextService.reload()));
  }
}
