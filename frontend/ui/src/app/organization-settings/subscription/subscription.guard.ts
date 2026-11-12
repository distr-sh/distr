import {inject} from '@angular/core';
import {CanActivateChildFn, Router} from '@angular/router';
import {firstValueFrom, map, Observable, switchMap} from 'rxjs';
import {AuthService} from '../../services/auth.service';
import {OrganizationService} from '../../services/organization.service';
import {isSubscriptionExpired} from '../../types/organization';
import {canOpenSettingsTab} from '../organization-settings-navigation';

export const subscriptionPath = '/settings/organization/subscription';

export function canOpenSubscription(): Observable<boolean> {
  const auth = inject(AuthService);
  return canOpenSettingsTab('subscription').pipe(map((canOpen) => auth.isVendor() && canOpen));
}

// canActivate would not run again when navigating between children.
export const subscriptionGuard: CanActivateChildFn = (_, state) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const canOpenSubscription$ = canOpenSubscription();
  return inject(OrganizationService)
    .get()
    .pipe(
      switchMap(async (org) => {
        if (!isSubscriptionExpired(org)) {
          return true;
        } else if (auth.isLens()) {
          // a lens cannot reach the subscription page, so the session has to return to the vendor portal first
          await auth.switchContext(org.id!);
          return false;
        } else if (state.url.startsWith(subscriptionPath)) {
          return true;
        } else if (await firstValueFrom(canOpenSubscription$)) {
          return router.createUrlTree([subscriptionPath]);
        } else {
          return router.createUrlTree(['/unavailable']);
        }
      })
    );
};
