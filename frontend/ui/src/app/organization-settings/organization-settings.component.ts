import {CdkConnectedOverlay} from '@angular/cdk/overlay';
import {ChangeDetectionStrategy, Component, computed, inject, signal} from '@angular/core';
import {PlanBadgeComponent} from '../components/plan-badge.component';
import {PlanFeatureHintComponent} from '../components/plan-feature-hint.component';
import {TabBarComponent, TabItem} from '../components/tab-bar.component';
import {AuthService} from '../services/auth.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {CustomEmailComponent} from './custom-email.component';
import {CustomOidcComponent} from './custom-oidc.component';
import {GeneralSettingsComponent} from './general-settings.component';

type OrganizationSettingsTab = 'general' | 'email' | 'identity-provider';

@Component({
  selector: 'app-organization-settings',
  templateUrl: './organization-settings.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    CdkConnectedOverlay,
    TabBarComponent,
    PlanBadgeComponent,
    PlanFeatureHintComponent,
    GeneralSettingsComponent,
    CustomEmailComponent,
    CustomOidcComponent,
  ],
})
export class OrganizationSettingsComponent {
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly auth = inject(AuthService);

  private readonly vendorAdmin = computed(() => this.auth.isVendor() && this.auth.hasRole('admin'));

  protected readonly customEmailVisible = computed(
    () => this.vendorAdmin() && this.featureFlags.isCustomEmailsEnabled()
  );

  protected readonly customOidcVisible = computed(
    () => this.vendorAdmin() && this.featureFlags.isCustomOidcProvidersEnabled()
  );

  protected readonly tabs = computed<TabItem<OrganizationSettingsTab>[]>(() => {
    const tabs: TabItem<OrganizationSettingsTab>[] = [{id: 'general', label: 'General'}];
    if (this.vendorAdmin()) {
      // Without the feature the tab is shown as disabled rather than hidden, so that admins can
      // see that the feature exists and which plan it needs.
      tabs.push({id: 'email', label: 'Custom Email Sending Provider', disabled: !this.customEmailVisible()});
      tabs.push({
        id: 'identity-provider',
        label: 'Custom Identity Provider',
        disabled: !this.customOidcVisible(),
      });
    }
    return tabs;
  });

  protected readonly activeTab = signal<OrganizationSettingsTab>('general');
  protected readonly planHintOpen = signal(false);

  protected onTabClick(tab: TabItem<OrganizationSettingsTab>) {
    if (tab.disabled) {
      this.planHintOpen.update((open) => !open);
    }
  }
}
