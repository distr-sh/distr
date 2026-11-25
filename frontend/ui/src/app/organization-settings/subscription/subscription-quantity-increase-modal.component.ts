import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {Component, inject, OnInit, signal} from '@angular/core';
import {Router} from '@angular/router';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayData, OverlayService} from '../../services/overlay.service';
import {SubscriptionService} from '../../services/subscription.service';
import {ToastService} from '../../services/toast.service';
import {isExpiredSubscription, SubscriptionInfo, subscriptionPrice, UNLIMITED_QTY} from '../../types/subscription';
import {PendingSubscriptionUpdate, SubscriptionUpdateModalComponent} from './subscription-update-modal.component';

type SubscriptionQuantityKind = 'user' | 'customer';

/** The update that increases the quantity by one, or `undefined` if the subscription cannot be updated in place. */
function quantityIncreaseUpdate(
  info: SubscriptionInfo,
  kind: SubscriptionQuantityKind
): PendingSubscriptionUpdate | undefined {
  if (
    (info.subscriptionType !== 'pro' && info.subscriptionType !== 'business') ||
    !info.hasStripeSubscription ||
    isExpiredSubscription(info.subscriptionType, info.subscriptionEndsAt)
  ) {
    return undefined;
  }

  // Distr rejects quantities below the current usage
  const userAccountQuantity =
    Math.max(info.subscriptionUserAccountQuantity, info.currentUserAccountCount) + (kind === 'user' ? 1 : 0);
  const customerOrganizationQuantity =
    Math.max(info.subscriptionCustomerOrganizationQuantity, info.currentCustomerOrganizationCount) +
    (kind === 'customer' ? 1 : 0);

  const maxCustomers = info.limits[info.subscriptionType].maxCustomerOrganizations;
  if (maxCustomers !== UNLIMITED_QTY && customerOrganizationQuantity > maxCustomers) {
    return undefined;
  }

  return {
    userAccountQuantity,
    customerOrganizationQuantity,
    oldPrice: subscriptionPrice(
      info.subscriptionType,
      info.subscriptionPeriod,
      info.subscriptionUserAccountQuantity,
      info.subscriptionCustomerOrganizationQuantity
    ),
    newPrice: subscriptionPrice(
      info.subscriptionType,
      info.subscriptionPeriod,
      userAccountQuantity,
      customerOrganizationQuantity
    ),
    subscriptionPeriod: info.subscriptionPeriod,
  };
}

/**
 * Closes with `true` once the quantity has been increased. Opens the subscription page instead where the
 * subscription cannot be updated in place.
 */
@Component({
  selector: 'app-subscription-quantity-increase-modal',
  templateUrl: './subscription-quantity-increase-modal.component.html',
  imports: [SubscriptionUpdateModalComponent],
})
export class SubscriptionQuantityIncreaseModalComponent implements OnInit {
  private readonly kind = inject(OverlayData) as SubscriptionQuantityKind;
  private readonly dialogRef = inject(DialogRef) as DialogRef<boolean>;
  private readonly subscriptionService = inject(SubscriptionService);
  private readonly organizationService = inject(OrganizationService);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);

  protected readonly pendingUpdate = signal<PendingSubscriptionUpdate | undefined>(undefined);

  async ngOnInit() {
    try {
      const update = quantityIncreaseUpdate(await firstValueFrom(this.subscriptionService.get()), this.kind);
      if (update) {
        this.pendingUpdate.set(update);
      } else {
        await this.dialogRef.dismiss();
        await this.router.navigate(['/settings/organization/subscription']);
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
      await this.dialogRef.dismiss();
    }
  }

  protected async confirm() {
    this.organizationService.reload();
    await this.dialogRef.close(true);
  }

  protected async dismiss() {
    await this.dialogRef.dismiss();
  }
}

/** Resolves to whether the quantity has been increased. */
export async function increaseSubscriptionQuantity(
  overlay: OverlayService,
  kind: SubscriptionQuantityKind
): Promise<boolean> {
  const dialogRef = overlay.showModal<boolean>(SubscriptionQuantityIncreaseModalComponent, {
    data: kind,
    hasBackdrop: true,
    backdropStyleOnly: true,
    positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
  });
  return (await firstValueFrom(dialogRef.result())) === true;
}
