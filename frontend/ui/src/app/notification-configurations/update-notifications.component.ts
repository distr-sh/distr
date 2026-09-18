import {Component, inject, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {ActivatedRoute, NavigationEnd, Router, RouterOutlet} from '@angular/router';
import {filter} from 'rxjs';
import {PageComponent} from '../components/page.component';
import {TabBarComponent, TabItem} from '../components/tab-bar.component';

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

  protected readonly tabs: TabItem<UpdateNotificationTab>[] = [
    {id: 'applications', label: 'Applications'},
    {id: 'artifacts', label: 'Artifacts'},
  ];

  protected readonly activeTab = signal(this.tabFromRoute());

  constructor() {
    this.router.events
      .pipe(
        filter((event) => event instanceof NavigationEnd),
        takeUntilDestroyed()
      )
      .subscribe(() => this.activeTab.set(this.tabFromRoute()));
  }

  protected onTabClick(tab: TabItem<UpdateNotificationTab>) {
    this.router.navigate([tab.id], {relativeTo: this.route});
  }

  private tabFromRoute(): UpdateNotificationTab {
    const path = this.route.snapshot.firstChild?.routeConfig?.path;
    return updateNotificationTabs.find((tab) => tab === path) ?? defaultTab;
  }
}
