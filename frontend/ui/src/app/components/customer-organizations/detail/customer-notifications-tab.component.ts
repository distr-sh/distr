import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {faBell, faCircleArrowUp, faClockRotateLeft} from '@fortawesome/free-solid-svg-icons';
import {EMPTY} from 'rxjs';
import {AlertConfigurationsComponent} from '../../../alert-configurations/alert-configurations.component';
import {UpdateNotificationsComponent} from '../../../notification-configurations/update-notifications.component';
import {NotificationRecordsComponent} from '../../../notification-records/notification-records.component';
import {CardHeadingComponent} from '../../card-heading.component';

@Component({
  templateUrl: './customer-notifications-tab.component.html',
  imports: [
    AlertConfigurationsComponent,
    CardHeadingComponent,
    NotificationRecordsComponent,
    UpdateNotificationsComponent,
  ],
  host: {class: 'flex flex-col gap-4'},
})
export class CustomerNotificationsTabComponent {
  protected readonly faBell = faBell;
  protected readonly faCircleArrowUp = faCircleArrowUp;
  protected readonly faClockRotateLeft = faClockRotateLeft;

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );
}
