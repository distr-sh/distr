import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBell, faCircleArrowUp, faClockRotateLeft} from '@fortawesome/free-solid-svg-icons';
import {AlertConfigurationsComponent} from '../alert-configurations/alert-configurations.component';
import {CardHeadingComponent} from '../components/card-heading.component';
import {PageComponent} from '../components/page.component';
import {SearchBarComponent} from '../components/search-bar.component';
import {UpdateNotificationsComponent} from '../notification-configurations/update-notifications.component';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';

@Component({
  templateUrl: './notifications-page.component.html',
  imports: [
    AlertConfigurationsComponent,
    CardHeadingComponent,
    FaIconComponent,
    PageComponent,
    RouterLink,
    SearchBarComponent,
    UpdateNotificationsComponent,
  ],
})
export class NotificationsPageComponent {
  protected readonly faBell = faBell;
  protected readonly faCircleArrowUp = faCircleArrowUp;
  protected readonly faClockRotateLeft = faClockRotateLeft;

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
