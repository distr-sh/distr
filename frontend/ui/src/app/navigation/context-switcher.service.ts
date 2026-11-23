import {computed, inject, Injectable, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {LensesService} from '../services/lenses.service';
import {ToastService} from '../services/toast.service';
import {CustomerLens} from '../types/lens';
import {Organization, OrganizationWithUserRole} from '../types/organization';
import {subscriptionRank} from '../types/subscription';

export interface SwitcherLens extends CustomerLens {
  current: boolean;
}

export interface SwitcherOrganization {
  org: OrganizationWithUserRole;
  current: boolean;
  lenses: SwitcherLens[];
}

/**
 * The organizations and customer lenses the session can switch to, for the navbar switcher and the command palette.
 * Provided on the logged-in shell like the NavigationService, since it reads the session's claims once.
 */
@Injectable()
export class ContextSwitcherService {
  private readonly auth = inject(AuthService);
  private readonly ctx = inject(ContextService);
  private readonly lensesService = inject(LensesService);
  private readonly toast = inject(ToastService);

  private readonly isLens = this.auth.isLens();
  private readonly allOrgs = toSignal(this.ctx.getAvailableOrganizations(), {initialValue: []});
  private readonly currentOrg = toSignal(this.ctx.getOrganization());
  private readonly lenses = signal<CustomerLens[]>([]);

  // The current organization comes first since it holds the lenses, which only it can enter.
  public readonly switcherTree = computed<SwitcherOrganization[]>(() => {
    const current = this.currentOrg();
    const lenses = this.lenses();
    return [...this.allOrgs()]
      .sort(
        (a, b) =>
          Number(b.id === current?.id) - Number(a.id === current?.id) ||
          subscriptionRank(a.subscriptionType, a.subscriptionEndsAt) -
            subscriptionRank(b.subscriptionType, b.subscriptionEndsAt) ||
          a.name.localeCompare(b.name)
      )
      .map((org) => {
        const isCurrentOrg = org.id === current?.id;
        return {
          org,
          current: isCurrentOrg && !this.isLens,
          lenses: isCurrentOrg
            ? lenses.map((lens) => ({
                ...lens,
                current: lens.customerOrganizationId === current?.customerOrganizationId,
              }))
            : [],
        };
      });
  });
  public readonly isSingleOrganization = computed(() => this.switcherTree().length === 1);
  // A single organization is left out of the list unless the session is in a lens, which it leads back from.
  public readonly isOrganizationListed = computed(() => !this.isSingleOrganization() || this.isLens);
  /** The membership that a session in a lens returns to. */
  public readonly lensHome = computed(() =>
    this.isLens ? this.allOrgs().find((org) => org.id === this.currentOrg()?.id) : undefined
  );

  constructor() {
    this.refreshLenses();
  }

  public async refreshLenses(): Promise<void> {
    try {
      this.lenses.set(await firstValueFrom(this.lensesService.list()));
    } catch {
      // the switcher still offers the organizations without the lenses
    }
  }

  public async switchOrganization(org: Organization): Promise<void> {
    await this.switchContextOrEnterLens(() => this.auth.switchContext(org.id!));
  }

  public async leaveLens(): Promise<void> {
    const home = this.lensHome();
    if (home) {
      await this.switchOrganization(home);
    }
  }

  public async enterLens(lens: CustomerLens): Promise<void> {
    await this.switchContextOrEnterLens(() => this.auth.enterLens(lens.customerOrganizationId));
  }

  private async switchContextOrEnterLens(switchFn: () => Promise<void>) {
    try {
      await switchFn();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}
