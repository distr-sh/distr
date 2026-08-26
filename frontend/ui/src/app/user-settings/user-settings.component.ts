import {ChangeDetectionStrategy, Component, inject, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {ActivatedRoute, NavigationEnd, Router, RouterOutlet} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleExclamation} from '@fortawesome/free-solid-svg-icons';
import {filter} from 'rxjs';
import {PageComponent} from '../components/page.component';
import {TabBarComponent, TabItem} from '../components/tab-bar.component';
import {AuthService} from '../services/auth.service';

const userSettingsTabs = ['general', 'security', 'identities'] as const;
const defaultTab: UserSettingsTab = 'general';

type UserSettingsTab = (typeof userSettingsTabs)[number];

@Component({
  templateUrl: './user-settings.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent, TabBarComponent, RouterOutlet, PageComponent],
})
export class UserSettingsComponent {
  protected readonly faCircleExclamation = faCircleExclamation;

  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  // A session cannot become one, or stop being one, without a new login, so this is read once.
  protected readonly customOidcSession = this.auth.isCustomOidcSession();

  protected readonly tabs: TabItem<UserSettingsTab>[] = [
    {id: 'general', label: 'General'},
    {id: 'security', label: 'Security'},
    {id: 'identities', label: 'Connected identities'},
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

  protected onTabClick(tab: TabItem<UserSettingsTab>) {
    this.router.navigate([tab.id], {relativeTo: this.route});
  }

  private tabFromRoute(): UserSettingsTab {
    const path = this.route.snapshot.firstChild?.routeConfig?.path;
    return userSettingsTabs.find((tab) => tab === path) ?? defaultTab;
  }
}
