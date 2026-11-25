import {Component, inject} from '@angular/core';
import {ClosableDialog} from '../../components/confirm-dialog/closable-dialog';
import {OverlayData} from '../../services/overlay.service';
import {PendingSubscriptionUpdate, SubscriptionUpdateModalComponent} from './subscription-update-modal.component';

/** Closes with `true` once the subscription has been updated. */
@Component({
  selector: 'app-subscription-quantity-increase-modal',
  templateUrl: './subscription-quantity-increase-modal.component.html',
  imports: [SubscriptionUpdateModalComponent],
})
export class SubscriptionQuantityIncreaseModalComponent extends ClosableDialog<boolean> {
  protected readonly pendingUpdate = inject(OverlayData) as PendingSubscriptionUpdate;

  protected async confirm() {
    await this.dialogRef.close(true);
  }

  protected async dismiss() {
    await this.dialogRef.dismiss();
  }
}
