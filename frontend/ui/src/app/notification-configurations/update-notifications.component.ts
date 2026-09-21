import {Component, computed, effect, inject, signal} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, NavigationEnd, Router, RouterOutlet} from '@angular/router';
import {filter, map} from 'rxjs';
import {PageComponent} from '../components/page.component';
import {TabBarComponent, TabItem} from '../components/tab-bar.component';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';

const updateNotificationTabs = ['applications', 'artifacts'] as const;
const defaultTab: UpdateNotificationTab = 'applications';

type UpdateNotificationTab = (typeof updateNotificationTabs)[number];

@Component({
  templateUrl: './update-notifications.component.html',
  imports: [TabBarComponent, RouterOutlet, PageComponent],
})
export class UpdateNotificationsComponent {
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly auth = inject(AuthService);
  private readonly context = inject(ContextService);

  private readonly customerFeatures = toSignal(
    this.context.getCustomerOrganization().pipe(map((customer) => customer?.features ?? [])),
    {initialValue: []}
  );

  // A customer watches only what it can reach: applications through its deployments, artifacts
  // through the registry.
  protected readonly tabs = computed<TabItem<UpdateNotificationTab>[]>(() => {
    const features = this.customerFeatures();
    const isCustomer = this.auth.isCustomer();
    return [
      {id: 'applications' as const, label: 'Applications', visible: features.includes('deployment_targets')},
      {id: 'artifacts' as const, label: 'Artifacts', visible: features.includes('artifacts')},
    ]
      .filter((tab) => !isCustomer || tab.visible)
      .map(({id, label}) => ({id, label}));
  });

  protected readonly activeTab = signal(this.tabFromRoute());

  constructor() {
    this.router.events
      .pipe(
        filter((event) => event instanceof NavigationEnd),
        takeUntilDestroyed()
      )
      .subscribe(() => this.activeTab.set(this.tabFromRoute()));

    effect(() => {
      const tabs = this.tabs();
      if (tabs.length > 0 && !tabs.some((tab) => tab.id === this.activeTab())) {
        this.router.navigate([tabs[0].id], {relativeTo: this.route, replaceUrl: true});
      }
    });
  }

  protected onTabClick(tab: TabItem<UpdateNotificationTab>) {
    this.router.navigate([tab.id], {relativeTo: this.route});
  }

  private tabFromRoute(): UpdateNotificationTab {
    const path = this.route.snapshot.firstChild?.routeConfig?.path;
    return updateNotificationTabs.find((tab) => tab === path) ?? defaultTab;
  }
}
