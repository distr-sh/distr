import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe, NgTemplateOutlet} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, computed, inject, input, OnInit, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArrowLeft,
  faBarsStaggered,
  faCheck,
  faChevronDown,
  faGlasses,
  faShuffle,
  faUserCircle,
} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, lastValueFrom, map, of} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {MembershipLabelPipe, organizationKind} from '../../../util/organization-kind';
import {SecureImagePipe} from '../../../util/secureImage';
import {UserRoleLabelPipe} from '../../../util/user-role';
import {ContextSwitcherService, SwitcherLens, SwitcherOrganization} from '../../navigation/context-switcher.service';
import {AuthService} from '../../services/auth.service';
import {ContextService} from '../../services/context.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {OrganizationService} from '../../services/organization.service';
import {SidebarService} from '../../services/sidebar.service';
import {ToastService} from '../../services/toast.service';
import {UsersService} from '../../services/users.service';
import {Organization} from '../../types/organization';
import {AvatarComponent} from '../avatar.component';
import {ColorSchemeSwitcherComponent} from '../color-scheme-switcher/color-scheme-switcher.component';
import {SearchBarComponent} from '../search-bar.component';
import {SubscriptionBadgeComponent} from '../subscription-badge.component';
import {NavBarLensBannerComponent} from './nav-bar-lens-banner/nav-bar-lens-banner.component';
import {NavBarSubscriptionBannerComponent} from './nav-bar-subscription-banner/nav-bar-subscription-banner.component';

// above this many entries the dropdown gets a search field
const ORGANIZATION_SEARCH_THRESHOLD = 4;

@Component({
  selector: 'app-nav-bar',
  templateUrl: './nav-bar.component.html',
  host: {
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
    NgTemplateOutlet,
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
  private readonly switcher = inject(ContextSwitcherService);
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

  private readonly switcherTree = this.switcher.switcherTree;
  private readonly isSingleOrganization = this.switcher.isSingleOrganization;
  protected readonly isOrganizationListed = this.switcher.isOrganizationListed;
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

  protected readonly organizationSearch = new FormControl<string>('', {nonNullable: true});
  private readonly organizationSearchTerm = toSignal(
    this.organizationSearch.valueChanges.pipe(map((term) => term.trim().toLowerCase())),
    {initialValue: ''}
  );
  protected readonly isOrganizationSearchVisible = computed(
    () => this.switchEntryCount() > ORGANIZATION_SEARCH_THRESHOLD
  );
  // A filter keeps the matching lenses and an organization that matches by name without its lenses.
  protected readonly switcherEntries = computed<SwitcherOrganization[]>(() => {
    const tree = this.switcherTree();
    const term = this.organizationSearchTerm();
    if (!term) {
      return tree;
    }
    return tree.flatMap((entry) => {
      const lenses = entry.lenses.filter((lens) => lens.customerOrganizationName.toLowerCase().includes(term));
      const orgMatches = this.isOrganizationListed() && entry.org.name.toLowerCase().includes(term);
      return lenses.length > 0 || orgMatches ? [{...entry, lenses}] : [];
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
      this.switcher.refreshLenses();
    }
    this.organizationsOpened.update((opened) => !opened);
  }

  protected async switchOrganization(org: Organization) {
    this.organizationsOpened.set(false);
    await this.switcher.switchOrganization(org);
  }

  protected async enterLens(lens: SwitcherLens) {
    this.organizationsOpened.set(false);
    await this.switcher.enterLens(lens);
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
  protected readonly faGlasses = faGlasses;
  protected readonly faShuffle = faShuffle;
  protected readonly faUserCircle = faUserCircle;
}
