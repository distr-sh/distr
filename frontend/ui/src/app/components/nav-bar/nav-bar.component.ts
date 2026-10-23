import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, computed, inject, input, linkedSignal, OnInit, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowLeft, faBarsStaggered, faCheck, faChevronDown, faUserCircle} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, lastValueFrom, map, of, startWith, Subject, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {MembershipLabelPipe, organizationKind} from '../../../util/organization-kind';
import {SecureImagePipe} from '../../../util/secureImage';
import {UserRoleLabelPipe} from '../../../util/user-role';
import {AuthService} from '../../services/auth.service';
import {ContextService} from '../../services/context.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {OrganizationService} from '../../services/organization.service';
import {SidebarService} from '../../services/sidebar.service';
import {ToastService} from '../../services/toast.service';
import {UsersService} from '../../services/users.service';
import {CustomerLens} from '../../types/lens';
import {Organization, OrganizationWithUserRole} from '../../types/organization';
import {subscriptionRank} from '../../types/subscription';
import {AvatarComponent} from '../avatar.component';
import {ColorSchemeSwitcherComponent} from '../color-scheme-switcher/color-scheme-switcher.component';
import {SearchBarComponent} from '../search-bar.component';
import {SubscriptionBadgeComponent} from '../subscription-badge.component';
import {NavBarLensBannerComponent} from './nav-bar-lens-banner/nav-bar-lens-banner.component';
import {NavBarSubscriptionBannerComponent} from './nav-bar-subscription-banner/nav-bar-subscription-banner.component';

// above this many entries the dropdown gets a search field
const ORGANIZATION_SEARCH_THRESHOLD = 4;

interface SwitcherLens extends CustomerLens {
  current: boolean;
}

interface SwitcherOrganization {
  org: OrganizationWithUserRole;
  current: boolean;
  lenses: SwitcherLens[];
}

interface SwitcherEntry extends SwitcherOrganization {
  collapsible: boolean;
  expanded: boolean;
}

@Component({
  selector: 'app-nav-bar',
  templateUrl: './nav-bar.component.html',
  host: {
    '(document:keydown.control.k)': 'onSwitcherShortcut($event)',
    '(document:keydown.meta.k)': 'onSwitcherShortcut($event)',
    '(document:keydown.escape)': 'organizationsOpened.set(false)',
  },
  imports: [
    AvatarComponent,
    ColorSchemeSwitcherComponent,
    NavBarLensBannerComponent,
    NavBarSubscriptionBannerComponent,
    OverlayModule,
    FaIconComponent,
    RouterLink,
    SearchBarComponent,
    SecureImagePipe,
    SubscriptionBadgeComponent,
    AsyncPipe,
    UserRoleLabelPipe,
    MembershipLabelPipe,
  ],
})
export class NavBarComponent implements OnInit {
  protected readonly auth = inject(AuthService);
  public readonly sidebar = inject(SidebarService);
  private readonly toast = inject(ToastService);
  private readonly route = inject(ActivatedRoute);
  private readonly usersService = inject(UsersService);
  private readonly organizationService = inject(OrganizationService);
  private readonly organizationBranding = inject(OrganizationBrandingService);
  private readonly ctx = inject(ContextService);
  protected readonly user$ = this.usersService.get().pipe(
    catchError(() => {
      const claims = this.auth.getClaims();
      if (claims) {
        return of({
          id: claims.sub,
          name: claims.name,
          email: claims.email,
          userRole: claims.role,
          imageUrl: claims.image_url,
        });
      }
      return EMPTY;
    })
  );

  protected readonly allOrgs = toSignal(this.ctx.getAvailableOrganizations(), {initialValue: []});
  protected readonly currentOrg = toSignal(this.ctx.getOrganization());
  protected readonly isDifferentOrganizationKind = computed(
    () => this.allOrgs().reduce((kinds, org) => kinds.add(organizationKind(org)), new Set<string>()).size > 1
  );
  protected readonly isTrial = computed(() => this.currentOrg()?.subscriptionType === 'trial');
  protected readonly isSubscriptionExpired = this.organizationService.isSubscriptionExpired;

  protected readonly isLens = this.auth.isLens();
  protected readonly isSuperAdmin = this.auth.isSuperAdmin();
  protected readonly ownKind = computed(() => {
    const membership = this.allOrgs().find((org) => org.id === this.currentOrg()?.id);
    return membership ? organizationKind(membership) : 'vendor';
  });

  private readonly lensesRefresh$ = new Subject<void>();
  private readonly lenses = toSignal(
    this.lensesRefresh$.pipe(
      startWith(undefined),
      switchMap(() => this.ctx.getLenses().pipe(catchError(() => EMPTY)))
    ),
    {initialValue: []}
  );
  // The current organization comes first since it holds the lenses, which only it can enter.
  protected readonly switcherTree = computed<SwitcherOrganization[]>(() => {
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
            ? lenses.map((lens) => ({...lens, current: lens.id === current?.customerOrganizationId}))
            : [],
        };
      });
  });
  protected readonly isSingleOrganization = computed(() => this.switcherTree().length === 1);
  // A single organization is left out of the list unless the session is in a lens, which it leads back from.
  protected readonly isOrganizationListed = computed(() => !this.isSingleOrganization() || this.isLens);
  protected readonly switchEntryCount = computed(() =>
    this.switcherTree().reduce(
      (count, entry) =>
        count +
        Number(!entry.current && this.isOrganizationListed()) +
        entry.lenses.filter((lens) => !lens.current).length,
      0
    )
  );
  protected readonly switcherFilterLabels = computed(() =>
    this.isSingleOrganization()
      ? {placeholder: 'Filter customers', empty: 'No customer matches your filter.'}
      : {placeholder: 'Filter organizations', empty: 'No organization matches your filter.'}
  );

  protected readonly userOpened = signal(false);
  protected readonly organizationsOpened = signal(false);
  protected readonly canOpenSwitcher = computed(() => this.switchEntryCount() > 0);
  protected readonly switcherShortcut = navigator.userAgent.includes('Mac') ? '⌘K' : 'Ctrl+K';

  protected readonly organizationSearch = new FormControl<string>('', {nonNullable: true});
  private readonly organizationSearchTerm = toSignal(
    this.organizationSearch.valueChanges.pipe(map((term) => term.trim().toLowerCase())),
    {initialValue: ''}
  );
  protected readonly isOrganizationSearchVisible = computed(
    () => this.switchEntryCount() > ORGANIZATION_SEARCH_THRESHOLD
  );
  // What the user expanded or collapsed, which a new filter discards since it changes what is expanded by default.
  private readonly switcherExpansion = linkedSignal<string, ReadonlyMap<string, boolean>>({
    source: this.organizationSearchTerm,
    computation: () => new Map(),
  });
  // With several organizations the lenses start collapsed unless the session is in one of them. A filter shows the
  // matching lenses expanded and all lenses collapsed when only their organization matches by name.
  protected readonly switcherEntries = computed<SwitcherEntry[]>(() => {
    const tree = this.switcherTree();
    const term = this.organizationSearchTerm();
    const expansion = this.switcherExpansion();
    const collapsible = tree.length > 1;
    return tree.flatMap((entry) => {
      let lenses = entry.lenses;
      let expandedByDefault = lenses.some((lens) => lens.current);
      if (term) {
        const matchingLenses = lenses.filter((lens) => lens.name.toLowerCase().includes(term));
        if (matchingLenses.length > 0) {
          lenses = matchingLenses;
          expandedByDefault = true;
        } else if (collapsible && entry.org.name.toLowerCase().includes(term)) {
          expandedByDefault = false;
        } else {
          return [];
        }
      }
      const hasToggle = collapsible && lenses.length > 0;
      return [
        {
          ...entry,
          lenses,
          collapsible: hasToggle,
          expanded: !hasToggle || (expansion.get(entry.org.id!) ?? expandedByDefault),
        },
      ];
    });
  });

  // Derived from the branding service's reactive stream so the navbar updates live when the logo/title is saved.
  private readonly branding = toSignal(this.organizationBranding.branding$);
  protected readonly logoUrl = computed(() => this.branding()?.logoImageId ?? '/distr-logo.svg');
  protected readonly hasCustomLogo = computed(() => !!this.branding()?.logoImageId);
  protected readonly customerSubtitle = computed(() => this.branding()?.title || 'Customer Portal');

  protected readonly faBarsStaggered = faBarsStaggered;
  protected readonly tutorial = toSignal(this.route.queryParams.pipe(map((params) => params['tutorial'])));

  public readonly isSubscriptionBannerVisible = input<boolean>();
  public readonly isSidebarVisible = input<boolean>();

  public async ngOnInit() {
    // Trigger the initial load; the template updates reactively via the branding service's stream.
    try {
      await lastValueFrom(this.organizationBranding.get());
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg && e instanceof HttpErrorResponse && e.status !== 404) {
        this.toast.error(msg);
      }
    }
  }

  protected toggleOrganizations() {
    this.organizationSearch.setValue('');
    if (!this.organizationsOpened()) {
      this.lensesRefresh$.next();
    }
    this.organizationsOpened.update((opened) => !opened);
  }

  protected onSwitcherShortcut(event: Event) {
    if (!this.canOpenSwitcher()) {
      return;
    }
    // Ctrl+K and Cmd+K are the browsers' own shortcuts for their search and address bars
    event.preventDefault();
    this.toggleOrganizations();
  }

  protected toggleSwitcherEntry(entry: SwitcherEntry) {
    this.switcherExpansion.update((expansion) => new Map(expansion).set(entry.org.id!, !entry.expanded));
  }

  protected async switchOrganization(org: Organization) {
    await this.switchContextOrEnterLens(() => this.auth.switchContext(org.id!));
  }

  protected async enterLens(lens: SwitcherLens) {
    await this.switchContextOrEnterLens(() => this.auth.enterLens(lens.id));
  }

  private async switchContextOrEnterLens(switchFn: () => Promise<void>) {
    this.organizationsOpened.set(false);
    try {
      await switchFn();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  async logout() {
    await lastValueFrom(this.auth.logout());
    // This is necessary to flush the caching crud services
    // TODO: implement flushing of services directly and switch to router.navigate(...)
    location.assign('/login');
  }

  protected readonly faArrowLeft = faArrowLeft;
  protected readonly faCheck = faCheck;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faUserCircle = faUserCircle;
}
