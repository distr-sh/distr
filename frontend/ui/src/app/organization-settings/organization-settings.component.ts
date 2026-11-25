import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {PageComponent} from '../components/page.component';
import {RoutedTabsComponent} from '../components/routed-tabs.component';
import {tabContext} from '../navigation/tabs';
import {OrganizationService} from '../services/organization.service';
import {organizationSettingsTabs} from './organization-settings-navigation';

@Component({
  selector: 'app-organization-settings',
  templateUrl: './organization-settings.component.html',
  imports: [PageComponent, RoutedTabsComponent],
})
export class OrganizationSettingsComponent {
  private readonly organizationService = inject(OrganizationService);
  private readonly context = toSignal(tabContext());

  protected readonly tabs = computed(() => {
    const context = this.context();
    if (context === undefined) {
      return [];
    }
    const tabs = organizationSettingsTabs.resolve(context);
    return this.organizationService.isSubscriptionExpired() ? tabs.filter((tab) => tab.id === 'subscription') : tabs;
  });
}
