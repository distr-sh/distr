import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {inject, Injectable} from '@angular/core';
import {Router} from '@angular/router';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {SubscriptionQuantityIncreaseModalComponent} from '../organization-settings/subscription/subscription-quantity-increase-modal.component';
import {PendingSubscriptionUpdate} from '../organization-settings/subscription/subscription-update-modal.component';
import {isExpiredSubscription, SubscriptionInfo, subscriptionPrice, UNLIMITED_QTY} from '../types/subscription';
import {OrganizationService} from './organization.service';
import {OverlayService} from './overlay.service';
import {SubscriptionService} from './subscription.service';
import {ToastService} from './toast.service';

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

@Injectable({providedIn: 'root'})
export class SubscriptionQuantityService {
  private readonly subscriptionService = inject(SubscriptionService);
  private readonly organizationService = inject(OrganizationService);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);

  /**
   * Resolves to whether the quantity has been increased. Opens the subscription page instead where the subscription
   * cannot be updated in place.
   */
  public async increase(overlay: OverlayService, kind: SubscriptionQuantityKind): Promise<boolean> {
    let update: PendingSubscriptionUpdate | undefined;
    try {
      update = quantityIncreaseUpdate(await firstValueFrom(this.subscriptionService.get()), kind);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
      return false;
    }

    if (!update) {
      await this.router.navigate(['/settings/organization/subscription']);
      return false;
    }

    const dialogRef = overlay.showModal<boolean>(SubscriptionQuantityIncreaseModalComponent, {
      data: update,
      hasBackdrop: true,
      backdropStyleOnly: true,
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
    const isIncreased = (await firstValueFrom(dialogRef.result())) === true;
    if (isIncreased) {
      this.organizationService.reload();
    }
    return isIncreased;
  }
}
