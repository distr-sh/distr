import {Component, computed, inject, TemplateRef, viewChild} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClockRotateLeft, faXmark} from '@fortawesome/free-solid-svg-icons';
import {EMPTY, switchMap} from 'rxjs';
import {NotificationRecordsComponent} from '../../../notification-records/notification-records.component';
import {NotificationsComponent} from '../../../notifications/notifications.component';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {NotificationRecordsService} from '../../../services/notification-records.service';
import {DialogRef, OverlayService} from '../../../services/overlay.service';
import {CustomerFeatureDisabledAlertComponent} from './customer-feature-disabled-alert.component';

@Component({
  templateUrl: './customer-notifications-tab.component.html',
  imports: [
    CustomerFeatureDisabledAlertComponent,
    FaIconComponent,
    NotificationRecordsComponent,
    NotificationsComponent,
  ],
  host: {class: 'flex flex-col gap-4'},
})
export class CustomerNotificationsTabComponent {
  protected readonly faClockRotateLeft = faClockRotateLeft;
  protected readonly faXmark = faXmark;

  private readonly overlay = inject(OverlayService);
  private readonly notificationRecordsService = inject(NotificationRecordsService);
  private readonly historyDrawer = viewChild.required<TemplateRef<unknown>>('historyDrawer');
  private historyDrawerRef?: DialogRef;

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );

  private readonly customerOrganizations = toSignal(inject(CustomerOrganizationsService).getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.customerOrganizationId())
  );
  protected readonly isAlertsDisabled = this.isFeatureDisabled('alerts');
  protected readonly isUpdateNotificationsDisabled = this.isFeatureDisabled('update_notifications');

  private readonly notificationRecords = toSignal(
    toObservable(this.customerOrganizationId).pipe(
      switchMap((customerOrganizationId) => this.notificationRecordsService.list(customerOrganizationId))
    )
  );
  protected readonly hasHistory = computed(() => !!this.notificationRecords()?.length);

  protected showHistory() {
    this.hideHistory();
    this.historyDrawerRef = this.overlay.showDrawer(this.historyDrawer());
  }

  protected hideHistory() {
    this.historyDrawerRef?.close();
  }

  private isFeatureDisabled(feature: CustomerOrganizationFeature) {
    return computed(() => {
      const customer = this.customerOrganization();
      return customer !== undefined && !customer.features.includes(feature);
    });
  }
}
