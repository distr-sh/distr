import {CdkConnectedOverlay} from '@angular/cdk/overlay';
import {ChangeDetectionStrategy, Component, computed, inject, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {ActivatedRoute, NavigationEnd, Router, RouterOutlet} from '@angular/router';
import {filter} from 'rxjs';
import {PlanBadgeComponent} from '../components/plan-badge.component';
import {PlanFeatureHintComponent} from '../components/plan-feature-hint.component';
import {TabBarComponent, TabItem} from '../components/tab-bar.component';
import {AuthService} from '../services/auth.service';
import {FeatureFlagService} from '../services/feature-flag.service';

const organizationSettingsTabs = ['general', 'identity-provider', 'email'] as const;
const defaultTab: OrganizationSettingsTab = 'general';

type OrganizationSettingsTab = (typeof organizationSettingsTabs)[number];

@Component({
  selector: 'app-organization-settings',
  templateUrl: './organization-settings.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [CdkConnectedOverlay, TabBarComponent, PlanBadgeComponent, PlanFeatureHintComponent, RouterOutlet],
})
export class OrganizationSettingsComponent {
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

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
      tabs.push({
        id: 'identity-provider',
        label: 'Identity Provider',
        disabled: !this.customOidcVisible(),
      });
      tabs.push({id: 'email', label: 'Email Sending Provider', disabled: !this.customEmailVisible()});
    }
    return tabs;
  });

  protected readonly activeTab = signal(this.tabFromRoute());
  protected readonly planHintTab = signal<OrganizationSettingsTab | null>(null);

  constructor() {
    this.router.events
      .pipe(
        filter((event) => event instanceof NavigationEnd),
        takeUntilDestroyed()
      )
      .subscribe(() => this.activeTab.set(this.tabFromRoute()));
  }

  protected onTabClick(tab: TabItem<OrganizationSettingsTab>) {
    if (tab.disabled) {
      this.planHintTab.update((open) => (open === tab.id ? null : tab.id));
    } else {
      this.router.navigate([tab.id], {relativeTo: this.route});
    }
  }

  private tabFromRoute(): OrganizationSettingsTab {
    const path = this.route.snapshot.firstChild?.routeConfig?.path;
    return organizationSettingsTabs.find((tab) => tab === path) ?? defaultTab;
  }
}
