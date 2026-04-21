import {CurrencyPipe} from '@angular/common';
import {Component, inject, input, output} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faXmark} from '@fortawesome/free-solid-svg-icons';
import {WEBSITE_URL} from '../../env/constants';
import {getFormDisplayedError} from '../../util/errors';
import {SubscriptionService} from '../services/subscription.service';
import {ToastService} from '../services/toast.service';
import {SubscriptionInfo, SubscriptionPeriod} from '../types/subscription';

export interface PendingSubscriptionUpdate {
  userAccountQuantity: number;
  customerOrganizationQuantity: number;
  newPrice: number;
  oldPrice: number;
  subscriptionPeriod: SubscriptionPeriod;
}

@Component({
  selector: 'app-subscription-update-modal',
  templateUrl: './subscription-update-modal.component.html',
  imports: [FaIconComponent, CurrencyPipe],
})
export class SubscriptionUpdateModalComponent {
  protected readonly xmarkIcon = faXmark;
  protected readonly checkIcon = faCheck;
  protected readonly websiteUrl = WEBSITE_URL;

  private readonly subscriptionService = inject(SubscriptionService);
  private readonly toast = inject(ToastService);

  protected editFormLoading = false;

  pendingUpdate = input.required<PendingSubscriptionUpdate>();
  closed = output<void>();
  confirmed = output<SubscriptionInfo>();

  async confirmUpdate() {
    this.editFormLoading = true;
    const pending = this.pendingUpdate();

    try {
      const body = {
        subscriptionUserAccountQuantity: pending.userAccountQuantity,
        subscriptionCustomerOrganizationQuantity: pending.customerOrganizationQuantity,
      };

      const updatedInfo = await this.subscriptionService.updateSubscription(body);
      this.toast.success('Subscription updated successfully');
      this.confirmed.emit(updatedInfo);
      setTimeout(() => location.reload(), 1000);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.editFormLoading = false;
    }
  }

  close() {
    this.closed.emit();
  }
}
