import {inject, Injectable} from '@angular/core';
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

  public readonly isNotificationsEnabled$ = this.requireSubscriptionType('trial', 'pro', 'enterprise');

  private requireSubscriptionType(...type: SubscriptionType[]) {
    return this.organizationService.get().pipe(map((org) => type.includes(org.subscriptionType)));
  }
}
