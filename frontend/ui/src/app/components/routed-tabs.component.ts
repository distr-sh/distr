import {CdkConnectedOverlay} from '@angular/cdk/overlay';
import {Component, inject, input, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {ActivatedRoute, NavigationEnd, Router, RouterOutlet} from '@angular/router';
import {filter} from 'rxjs';
import {LockableTab} from '../navigation/tabs';
import {PlanBadgeComponent} from './plan-badge.component';
import {PlanFeatureHintComponent} from './plan-feature-hint.component';
import {TabBarComponent} from './tab-bar.component';

@Component({
  selector: 'app-routed-tabs',
  templateUrl: './routed-tabs.component.html',
  host: {class: 'block'},
  imports: [CdkConnectedOverlay, TabBarComponent, PlanBadgeComponent, PlanFeatureHintComponent, RouterOutlet],
})
export class RoutedTabsComponent {
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  public readonly tabs = input.required<readonly LockableTab[]>();
  protected readonly active = signal(this.tabFromRoute());
  public readonly activeTab = this.active.asReadonly();
  protected readonly planHintTab = signal<string | null>(null);

  constructor() {
    this.router.events
      .pipe(
        filter((event) => event instanceof NavigationEnd),
        takeUntilDestroyed()
      )
      .subscribe(() => this.active.set(this.tabFromRoute()));
  }

  protected onTabClick(tab: LockableTab) {
    if (tab.disabled) {
      this.planHintTab.update((open) => (open === tab.id ? null : tab.id));
    } else {
      this.router.navigate([tab.id], {relativeTo: this.route});
    }
  }

  private tabFromRoute(): string {
    return this.route.snapshot.firstChild?.routeConfig?.path ?? '';
  }
}
