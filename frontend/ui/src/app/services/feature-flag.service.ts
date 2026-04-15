import {inject, Injectable} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {map} from 'rxjs';
import {SubscriptionType} from '../types/subscription';
import {OrganizationService} from './organization.service';

@Injectable({
  providedIn: 'root',
})
export class FeatureFlagService {
  private readonly organizationService = inject(OrganizationService);
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

  public readonly isNotificationsEnabled$ = this.requireSubscriptionType('trial', 'pro', 'enterprise');

  public readonly isSupportBundlesEnabled$ = this.requireSubscriptionType('trial', 'pro', 'enterprise');

  private requireSubscriptionType(...type: SubscriptionType[]) {
    return this.organizationService.get().pipe(map((org) => type.includes(org.subscriptionType)));
  }
}
