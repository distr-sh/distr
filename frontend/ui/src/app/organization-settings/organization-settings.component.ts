import {CdkConnectedOverlay} from '@angular/cdk/overlay';
import {ChangeDetectionStrategy, Component, computed, inject, signal} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, NavigationEnd, Router, RouterOutlet} from '@angular/router';
import {filter} from 'rxjs';
import {PageComponent} from '../components/page.component';
import {PlanBadgeComponent} from '../components/plan-badge.component';
import {PlanFeatureHintComponent} from '../components/plan-feature-hint.component';
import {TabBarComponent, TabItem} from '../components/tab-bar.component';
import {OrganizationService} from '../services/organization.service';
import {
  resolveSettingsTabs,
  settingsTabContext,
  SettingsTabId,
  settingsTabIds,
} from './organization-settings-navigation';

const defaultTab: SettingsTabId = 'general';

@Component({
  selector: 'app-organization-settings',
  templateUrl: './organization-settings.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    CdkConnectedOverlay,
    TabBarComponent,
    PlanBadgeComponent,
    PlanFeatureHintComponent,
    RouterOutlet,
    PageComponent,
  ],
})
export class OrganizationSettingsComponent {
  private readonly organizationService = inject(OrganizationService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  private readonly context = toSignal(settingsTabContext());

  protected readonly tabs = computed(() => {
    const context = this.context();
    if (context === undefined) {
      return [];
    }
    const tabs = resolveSettingsTabs(context);
    return this.organizationService.isSubscriptionExpired() ? tabs.filter((tab) => tab.id === 'subscription') : tabs;
  });

  protected readonly activeTab = signal(this.tabFromRoute());
  protected readonly planHintTab = signal<SettingsTabId | null>(null);

  constructor() {
    this.router.events
      .pipe(
        filter((event) => event instanceof NavigationEnd),
        takeUntilDestroyed()
      )
      .subscribe(() => this.activeTab.set(this.tabFromRoute()));
  }

  protected onTabClick(tab: TabItem<SettingsTabId>) {
    if (tab.disabled) {
      this.planHintTab.update((open) => (open === tab.id ? null : tab.id));
    } else {
      this.router.navigate([tab.id], {relativeTo: this.route});
    }
  }

  private tabFromRoute(): SettingsTabId {
    const path = this.route.snapshot.firstChild?.routeConfig?.path;
    return settingsTabIds.find((tab) => tab === path) ?? defaultTab;
  }
}
