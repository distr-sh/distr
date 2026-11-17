import {Component, computed, inject, input, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl} from '@angular/forms';
import {AlertConfigurationsComponent} from '../alert-configurations/alert-configurations.component';
import {CardHeadingComponent} from '../components/card-heading.component';
import {SearchBarComponent} from '../components/search-bar.component';
import {UpdateNotificationsComponent} from '../notification-configurations/update-notifications.component';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';

/** An element with `toolbarActions` is placed on the right of the toolbar. */
@Component({
  selector: 'app-notifications',
  templateUrl: './notifications.component.html',
  imports: [AlertConfigurationsComponent, CardHeadingComponent, SearchBarComponent, UpdateNotificationsComponent],
  host: {class: 'block'},
})
export class NotificationsComponent {
  public readonly customerOrganizationId = input<string>();
  public readonly alertsDisabled = input(false);
  public readonly updateNotificationsDisabled = input(false);

  private readonly alerts = viewChild(AlertConfigurationsComponent);
  private readonly updates = viewChild(UpdateNotificationsComponent);
  protected readonly hasVisiblePanel = computed(() =>
    [this.alerts(), this.updates()].some((panel) => panel !== undefined && !panel.isHidden())
  );

  protected readonly searchControl = new FormControl('', {nonNullable: true});
  protected readonly search = toSignal(this.searchControl.valueChanges, {initialValue: ''});

  private readonly auth = inject(AuthService);
  private readonly customerOrganization = toSignal(inject(ContextService).getCustomerOrganization());

  protected readonly alertsVisible = computed(
    () => !this.auth.isCustomer() || !!this.customerOrganization()?.features.includes('alerts')
  );
  protected readonly updatesVisible = computed(
    () => !this.auth.isCustomer() || !!this.customerOrganization()?.features.includes('update_notifications')
  );
}
