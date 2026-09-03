import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, computed, inject, input, OnInit, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArrowLeft,
  faBarsStaggered,
  faCheck,
  faCheckDouble,
  faChevronDown,
  faChevronUp,
  faCircleExclamation,
  faClipboard,
  faLightbulb,
  faPlus,
  faShuffle,
  faUserCircle,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, lastValueFrom, map, of} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {MembershipLabelPipe, organizationKind} from '../../../util/organization-kind';
import {SecureImagePipe} from '../../../util/secureImage';
import {UserRoleLabelPipe} from '../../../util/user-role';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AuthService} from '../../services/auth.service';
import {ContextService} from '../../services/context.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SidebarService} from '../../services/sidebar.service';
import {ToastService} from '../../services/toast.service';
import {UsersService} from '../../services/users.service';
import {Organization} from '../../types/organization';
import {subscriptionRank} from '../../types/subscription';
import {ColorSchemeSwitcherComponent} from '../color-scheme-switcher/color-scheme-switcher.component';
import {SearchBarComponent} from '../search-bar.component';
import {SubscriptionBadgeComponent} from '../subscription-badge.component';
import {NavBarSubscriptionBannerComponent} from './nav-bar-subscription-banner/nav-bar-subscription-banner.component';

// above this many switchable organizations the dropdown gets a search field
const ORGANIZATION_SEARCH_THRESHOLD = 4;

@Component({
  selector: 'app-nav-bar',
  templateUrl: './nav-bar.component.html',
  host: {
    '(document:keydown.control.k)': 'onSwitcherShortcut($event)',
    '(document:keydown.meta.k)': 'onSwitcherShortcut($event)',
    '(document:keydown.escape)': 'organizationsOpened.set(false)',
  },
  imports: [
    ColorSchemeSwitcherComponent,
    NavBarSubscriptionBannerComponent,
    OverlayModule,
    FaIconComponent,
    RouterLink,
    SearchBarComponent,
    SecureImagePipe,
    SubscriptionBadgeComponent,
    AsyncPipe,
    AutotrimDirective,
    ReactiveFormsModule,
    UserRoleLabelPipe,
    MembershipLabelPipe,
  ],
})
export class NavBarComponent implements OnInit {
  protected readonly auth = inject(AuthService);
  private readonly overlay = inject(OverlayService);
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
  protected readonly availableOrgs = computed(() => {
    const current = this.currentOrg();
    return this.allOrgs()
      .filter((org) => org.id !== current?.id)
      .sort(
        (a, b) =>
          subscriptionRank(a.subscriptionType, a.subscriptionEndsAt) -
            subscriptionRank(b.subscriptionType, b.subscriptionEndsAt) || a.name.localeCompare(b.name)
      );
  });
  protected readonly currentOrg = toSignal(this.ctx.getOrganization());
  protected readonly isVendorSomewhere = computed(() =>
    this.allOrgs().some((org) => organizationKind(org) === 'vendor')
  );
  protected readonly canCreateOrganization = toSignal(this.ctx.canCreateOrganization(), {initialValue: false});
  protected readonly isDifferentOrganizationKind = computed(
    () => this.allOrgs().reduce((kinds, org) => kinds.add(organizationKind(org)), new Set<string>()).size > 1
  );
  protected readonly isTrial = computed(() => this.currentOrg()?.subscriptionType === 'trial');
  protected readonly isSubscriptionExpired = this.organizationService.isSubscriptionExpired;

  protected readonly userOpened = signal(false);
  protected readonly organizationsOpened = signal(false);
  protected readonly isOrganizationSwitcherVisible = computed(
    () => this.isVendorSomewhere() || this.availableOrgs().length > 0
  );
  protected readonly switcherShortcut = navigator.userAgent.includes('Mac') ? '⌘K' : 'Ctrl+K';

  protected readonly organizationSearch = new FormControl<string>('', {nonNullable: true});
  private readonly organizationSearchTerm = toSignal(this.organizationSearch.valueChanges, {initialValue: ''});
  protected readonly isOrganizationSearchVisible = computed(
    () => this.availableOrgs().length > ORGANIZATION_SEARCH_THRESHOLD
  );
  protected readonly matchingOrgs = computed(() => {
    const term = this.organizationSearchTerm().trim().toLowerCase();
    const orgs = this.availableOrgs();
    return term ? orgs.filter((org) => org.name.toLowerCase().includes(term)) : orgs;
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

  private readonly createOrgModal = viewChild.required<TemplateRef<unknown>>('createOrgModal');
  private modalRef?: DialogRef;
  protected readonly createOrgForm = new FormGroup({
    name: new FormControl<string>('', Validators.required),
  });

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
    this.organizationsOpened.update((opened) => !opened);
  }

  protected onSwitcherShortcut(event: Event) {
    if (!this.isOrganizationSwitcherVisible() || this.modalRef) {
      return;
    }
    // Ctrl+K and Cmd+K are the browsers' own shortcuts for their search and address bars
    event.preventDefault();
    this.toggleOrganizations();
  }

  async switchContext(org: Organization, targetPath = '/') {
    this.organizationsOpened.set(false);
    try {
      const switched = await lastValueFrom(this.auth.switchContext(org));
      if (switched) {
        location.assign(targetPath);
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  showCreateOrgModal(): void {
    this.closeCreateOrgModal();
    this.organizationsOpened.set(false);
    this.modalRef = this.overlay.showModal(this.createOrgModal());
    this.modalRef.result().subscribe(() => (this.modalRef = undefined));
  }

  closeCreateOrgModal() {
    this.modalRef?.close();
    this.createOrgForm.reset();
  }

  async submitCreateOrgForm() {
    this.createOrgForm.markAllAsTouched();
    if (this.createOrgForm.valid) {
      try {
        const created = await lastValueFrom(this.organizationService.create(this.createOrgForm.value.name!));
        await this.switchContext(created, '/dashboard?from=new-org');
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      }
    }
  }

  async logout() {
    await lastValueFrom(this.auth.logout());
    // This is necessary to flush the caching crud services
    // TODO: implement flushing of services directly and switch to router.navigate(...)
    location.assign('/login');
  }

  protected readonly faLightbulb = faLightbulb;
  protected readonly faArrowLeft = faArrowLeft;
  protected readonly faShuffle = faShuffle;
  protected readonly faCheck = faCheck;
  protected readonly faCheckDouble = faCheckDouble;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faChevronUp = faChevronUp;
  protected readonly faPlus = faPlus;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faXmark = faXmark;
  protected readonly faClipboard = faClipboard;
  protected readonly faUserCircle = faUserCircle;
}
