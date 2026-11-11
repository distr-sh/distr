import {TestBed} from '@angular/core/testing';
import {ActivatedRouteSnapshot, provideRouter, Router, RouterStateSnapshot, UrlTree} from '@angular/router';
import {UserRole} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {firstValueFrom, Observable, of} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {OrganizationService} from '../services/organization.service';
import {Organization} from '../types/organization';
import {subscriptionGuard} from './subscription.guard';

describe('subscriptionGuard', () => {
  function setup(role: UserRole, subscriptionEndsAt: dayjs.Dayjs) {
    const organization: Pick<Organization, 'id' | 'subscriptionType' | 'subscriptionEndsAt' | 'features'> = {
      id: 'org',
      subscriptionType: 'pro',
      subscriptionEndsAt: subscriptionEndsAt.toISOString(),
      features: [],
    };
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {
            isLens: () => false,
            isVendor: () => true,
            isSuperAdmin: () => false,
            hasRole: (r: UserRole) => r === role,
          },
        },
        {provide: OrganizationService, useValue: {get: () => of(organization as Organization)}},
      ],
    });
  }

  async function navigate(url: string): Promise<string | boolean> {
    const result = TestBed.runInInjectionContext(() =>
      subscriptionGuard({} as ActivatedRouteSnapshot, {url} as RouterStateSnapshot)
    ) as Observable<boolean | UrlTree>;
    const outcome = await firstValueFrom(result);
    return outcome instanceof UrlTree ? TestBed.inject(Router).serializeUrl(outcome) : outcome;
  }

  it('sends an admin of an expired organization to the subscription tab', async () => {
    setup('admin', dayjs().subtract(1, 'day'));
    expect(await navigate('/dashboard')).toBe('/settings/organization/subscription');
  });

  it('lets an admin of an expired organization open the subscription tab and the checkout callback', async () => {
    setup('admin', dayjs().subtract(1, 'day'));
    expect(await navigate('/settings/organization/subscription')).toBe(true);
    expect(await navigate('/settings/organization/subscription/callback?pendingPlan=business')).toBe(true);
  });

  // The subscription tab is admin only, so sending a non-admin there would bounce between this guard
  // and the tab guard.
  it('sends a non-admin of an expired organization to the unavailable page', async () => {
    setup('read_write', dayjs().subtract(1, 'day'));
    expect(await navigate('/deployments')).toBe('/unavailable');
  });

  it('lets everyone through while the subscription is active', async () => {
    setup('read_only', dayjs().add(1, 'day'));
    expect(await navigate('/dashboard')).toBe(true);
  });
});
