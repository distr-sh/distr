import {CurrencyPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, input, output} from '@angular/core';
import {Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faXmark} from '@fortawesome/free-solid-svg-icons';
import {WEBSITE_URL} from '../../constants';
import {getFormDisplayedError} from '../../util/errors';
import {SubscriptionService} from '../services/subscription.service';
import {ToastService} from '../services/toast.service';
import {SubscriptionInfo, SubscriptionPeriod, SubscriptionType} from '../types/subscription';

export interface PendingSubscriptionUpdate {
  // Set when the update also switches the subscription to a different plan
  subscriptionType?: SubscriptionType;
  userAccountQuantity: number;
  customerOrganizationQuantity: number;
  newPrice: number;
  oldPrice: number;
  subscriptionPeriod: SubscriptionPeriod;
}

@Component({
  selector: 'app-subscription-update-modal',
  templateUrl: './subscription-update-modal.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent, CurrencyPipe],
})
export class SubscriptionUpdateModalComponent {
  protected readonly xmarkIcon = faXmark;
  protected readonly checkIcon = faCheck;
  protected readonly websiteUrl = WEBSITE_URL;

  private readonly subscriptionService = inject(SubscriptionService);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);

  protected editFormLoading = false;

  pendingUpdate = input.required<PendingSubscriptionUpdate>();
  closed = output<void>();
  confirmed = output<SubscriptionInfo>();

  async confirmUpdate() {
    this.editFormLoading = true;
    const pending = this.pendingUpdate();

    try {
      const body = {
        subscriptionType: pending.subscriptionType,
        subscriptionUserAccountQuantity: pending.userAccountQuantity,
        subscriptionCustomerOrganizationQuantity: pending.customerOrganizationQuantity,
      };

      const updatedInfo = await this.subscriptionService.updateSubscription(body);
      this.confirmed.emit(updatedInfo);
      if (pending.subscriptionType) {
        // Plan switches are only applied once Stripe confirms them via webhook,
        // so redirect to the callback page which waits for the confirmation.
        await this.router.navigate(['/subscription/callback'], {
          queryParams: {pendingPlan: pending.subscriptionType},
        });
      } else {
        this.toast.success('Subscription updated successfully');
        setTimeout(() => location.reload(), 1000);
      }
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
