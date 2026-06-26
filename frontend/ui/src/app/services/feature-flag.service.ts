import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {map, Observable, shareReplay} from 'rxjs';
import {ExperimentalFeatureFlag, ExperimentalFeatureFlagKey} from '../types/feature-flags';
import {SubscriptionType} from '../types/subscription';
import {OrganizationService} from './organization.service';

@Injectable({
  providedIn: 'root',
})
export class FeatureFlagService {
  private readonly httpClient = inject(HttpClient);
  private readonly organizationService = inject(OrganizationService);
  private readonly experimentalFeatureFlagsUrl = '/api/v1/experimental-feature-flags';
  private readonly experimentalFeatureFlags$ = this.httpClient
    .get<ExperimentalFeatureFlag[]>(this.experimentalFeatureFlagsUrl)
    .pipe(shareReplay(1));

  public readonly isLicensingEnabled$ = this.organizationService
    .get()
    .pipe(map((org) => org.features.includes('licensing')));
  public readonly isPrePostScriptEnabled$ = this.organizationService
    .get()
    .pipe(map((org) => org.features.includes('pre_post_scripts')));
  public readonly isVendorBillingEnabled$ = this.organizationService
    .get()
    .pipe(map((org) => org.features.includes('vendor_billing')));
  public readonly isVendorBillingEnabled = toSignal(this.isVendorBillingEnabled$, {initialValue: false});

  public readonly isDeploymentLogsAfterEnabled = toSignal(
    this.organizationService.get().pipe(map((org) => org.features.includes('deployment_logs_after'))),
    {initialValue: false}
  );

  public readonly isPartnerManagementEnabled$ = this.organizationService
    .get()
    .pipe(map((org) => org.features.includes('partner_management')));
  public readonly isPartnerManagementEnabled = toSignal(this.isPartnerManagementEnabled$, {initialValue: false});

  public readonly isNotificationsEnabled$ = this.requireSubscriptionType('trial', 'pro', 'enterprise');

  public readonly isSupportBundlesEnabled$ = this.requireSubscriptionType('trial', 'pro', 'enterprise');

  private requireSubscriptionType(...type: SubscriptionType[]) {
    return this.organizationService.get().pipe(map((org) => type.includes(org.subscriptionType)));
  }

  getExperimentalFeatureFlags(): Observable<ExperimentalFeatureFlag[]> {
    return this.experimentalFeatureFlags$;
  }

  public readonly isEnvironmentsEnabled$ = this.isExperimentalFeatureEnabled$('environments');

  isExperimentalFeatureEnabled$(key: ExperimentalFeatureFlagKey): Observable<boolean> {
    return this.getExperimentalFeatureFlags().pipe(
      map((flags) => flags.some((flag) => flag.key === key && flag.enabled))
    );
  }
}
