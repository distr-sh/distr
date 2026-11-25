import {Component, computed} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {PageComponent} from '../components/page.component';
import {RoutedTabsComponent} from '../components/routed-tabs.component';
import {tabContext} from '../navigation/tabs';
import {customerPortalTabs} from './customer-portal-navigation';

@Component({
  selector: 'app-customer-portal',
  templateUrl: './customer-portal.component.html',
  imports: [PageComponent, RoutedTabsComponent],
})
export class CustomerPortalComponent {
  private readonly context = toSignal(tabContext());

  protected readonly tabs = computed(() => {
    const context = this.context();
    return context === undefined ? [] : customerPortalTabs.resolve(context);
  });
}
