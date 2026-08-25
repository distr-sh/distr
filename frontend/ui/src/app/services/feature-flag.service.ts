import {inject, Injectable} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {distinctUntilChanged, map} from 'rxjs';
import {Feature} from '../types/organization';
import {NON_PRO_SUBSCRIPTION_TYPES, SubscriptionType} from '../types/subscription';
import {OrganizationService} from './organization.service';

@Injectable({
  providedIn: 'root',
})
export class FeatureFlagService {
  private readonly organizationService = inject(OrganizationService);
  public readonly isLicensingEnabled$ = this.hasFeature('licensing');
  public readonly isPrePostScriptEnabled$ = this.hasFeature('pre_post_scripts');
  public readonly isVendorBillingEnabled$ = this.hasFeature('vendor_billing');
  public readonly isVendorBillingEnabled = toSignal(this.isVendorBillingEnabled$, {initialValue: false});

  public readonly isDeploymentLogsAfterEnabled = toSignal(this.hasFeature('deployment_logs_after'), {
    initialValue: false,
  });

  public readonly isPartnerManagementEnabled$ = this.hasFeature('partner_management');
  public readonly isPartnerManagementEnabled = toSignal(this.isPartnerManagementEnabled$, {initialValue: false});

  public readonly isCustomDomainsEnabled$ = this.hasFeature('custom_domains');
  public readonly isCustomDomainsEnabled = toSignal(this.isCustomDomainsEnabled$, {initialValue: false});

  public readonly isCustomEmailsEnabled$ = this.hasFeature('custom_emails');
  public readonly isCustomEmailsEnabled = toSignal(this.isCustomEmailsEnabled$, {initialValue: false});

  public readonly isCustomOidcProvidersEnabled$ = this.hasFeature('custom_oidc_providers');
  public readonly isCustomOidcProvidersEnabled = toSignal(this.isCustomOidcProvidersEnabled$, {initialValue: false});

  public readonly isNotificationsEnabled$ = this.forbidSubscriptionType(...NON_PRO_SUBSCRIPTION_TYPES);

  public readonly isSupportBundlesEnabled$ = this.forbidSubscriptionType(...NON_PRO_SUBSCRIPTION_TYPES);

  // distinctUntilChanged because the organization is re-emitted on every context reload: without it,
  // every stream that combines a flag with something else re-runs, refetching lists that a reload of
  // some unrelated part of the context cannot have changed.
  private hasFeature(feature: Feature) {
    return this.organizationService.get().pipe(
      map((org) => org.features.includes(feature)),
      distinctUntilChanged()
    );
  }

  private forbidSubscriptionType(...type: SubscriptionType[]) {
    return this.organizationService.get().pipe(
      map((org) => !type.includes(org.subscriptionType)),
      distinctUntilChanged()
    );
  }
}
