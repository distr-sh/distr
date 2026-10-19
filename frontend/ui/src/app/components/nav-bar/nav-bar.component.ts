import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, computed, inject, input, linkedSignal, OnInit, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faAddressBook,
  faArrowLeft,
  faBarsStaggered,
  faBuilding,
  faCheck,
  faChevronDown,
  faPlus,
  faUserCircle,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, firstValueFrom, lastValueFrom, map, of, startWith, Subject, switchMap} from 'rxjs';
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
import {ContextSwitchPortal, ContextSwitchTargets, SwitchContextRequest} from '../../types/context-switch';
import {CreateOrganizationResponse, Organization, OrganizationWithUserRole} from '../../types/organization';
import {subscriptionRank} from '../../types/subscription';
import {AlertComponent} from '../alert/alert.component';
import {AvatarComponent} from '../avatar.component';
import {ClipComponent} from '../clip.component';
import {ColorSchemeSwitcherComponent} from '../color-scheme-switcher/color-scheme-switcher.component';
import {SearchBarComponent} from '../search-bar.component';
import {SubscriptionBadgeComponent} from '../subscription-badge.component';
import {NavBarForeignContextBannerComponent} from './nav-bar-foreign-context-banner/nav-bar-foreign-context-banner.component';
import {NavBarSubscriptionBannerComponent} from './nav-bar-subscription-banner/nav-bar-subscription-banner.component';

// above this many entries the dropdown gets a search field
const ORGANIZATION_SEARCH_THRESHOLD = 4;

const NO_SWITCH_TARGETS: ContextSwitchTargets = {organizations: []};

interface SwitcherPortal extends ContextSwitchPortal {
  kind: 'customer' | 'partner';
  current: boolean;
}

interface SwitcherOrganization {
  org: OrganizationWithUserRole;
  logoUrl?: string;
  current: boolean;
  portals: SwitcherPortal[];
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
    AlertComponent,
    AvatarComponent,
    ClipComponent,
    ColorSchemeSwitcherComponent,
    NavBarForeignContextBannerComponent,
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

  protected readonly isForeignContext = this.auth.isForeignContext();
  protected readonly isSuperAdmin = this.auth.isSuperAdmin();
  protected readonly ownPortalKind = computed(() => {
    const membership = this.allOrgs().find((org) => org.id === this.currentOrg()?.id);
    return membership ? organizationKind(membership) : 'vendor';
  });

  private readonly switchTargetsRefresh$ = new Subject<void>();
  private readonly switchTargets = toSignal(
    this.switchTargetsRefresh$.pipe(
      startWith(undefined),
      switchMap(() => this.ctx.getSwitchTargets().pipe(catchError(() => EMPTY)))
    ),
    {initialValue: NO_SWITCH_TARGETS}
  );
  // The current organization comes first since it holds the portal the session operates in, if any.
  protected readonly switcherTree = computed<SwitcherOrganization[]>(() => {
    const current = this.currentOrg();
    const targets = new Map(this.switchTargets().organizations.map((it) => [it.id, it]));
    return [...this.allOrgs()]
      .sort(
        (a, b) =>
          Number(b.id === current?.id) - Number(a.id === current?.id) ||
          subscriptionRank(a.subscriptionType, a.subscriptionEndsAt) -
            subscriptionRank(b.subscriptionType, b.subscriptionEndsAt) ||
          a.name.localeCompare(b.name)
      )
      .map((org) => {
        const target = targets.get(org.id!);
        const isCurrentOrg = org.id === current?.id;
        return {
          org,
          logoUrl: target?.logoUrl,
          current: isCurrentOrg && !this.isForeignContext,
          portals: [
            ...(target?.partnerOrganizations ?? []).map((portal) => ({
              ...portal,
              kind: 'partner' as const,
              current: isCurrentOrg && portal.id === current?.partnerOrganizationId,
            })),
            ...(target?.customerOrganizations ?? []).map((portal) => ({
              ...portal,
              kind: 'customer' as const,
              current: isCurrentOrg && portal.id === current?.customerOrganizationId,
            })),
          ],
        };
      });
  });
  // A single organization is left out of the list, the banner of a foreign context leads back to its own portal.
  protected readonly isSingleOrganization = computed(() => this.switcherTree().length === 1);
  protected readonly switchEntryCount = computed(() =>
    this.switcherTree().reduce(
      (count, entry) =>
        count +
        Number(!entry.current && !this.isSingleOrganization()) +
        entry.portals.filter((portal) => !portal.current).length,
      0
    )
  );
  protected readonly switcherFilterLabels = computed(() => {
    if (!this.isSingleOrganization()) {
      return {placeholder: 'Filter organizations', empty: 'No organization matches your filter.'};
    }
    const kinds = new Set(this.switcherTree()[0].portals.map((portal) => portal.kind));
    if (kinds.size > 1) {
      return {placeholder: 'Filter customers and partners', empty: 'No customer or partner matches your filter.'};
    } else if (kinds.has('partner')) {
      return {placeholder: 'Filter partners', empty: 'No partner matches your filter.'};
    }
    return {placeholder: 'Filter customers', empty: 'No customer matches your filter.'};
  });

  protected readonly userOpened = signal(false);
  protected readonly organizationsOpened = signal(false);
  protected readonly isOrganizationSwitcherVisible = computed(
    () => this.isVendorSomewhere() || this.isForeignContext || this.switchEntryCount() > 0
  );
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
  // With several organizations their portals start collapsed, except for the organization of the portal the session
  // operates in. A filter shows the matching portals expanded below their organization and all portals collapsed
  // below an organization that matches by name alone.
  protected readonly switcherEntries = computed<SwitcherEntry[]>(() => {
    const tree = this.switcherTree();
    const term = this.organizationSearchTerm();
    const expansion = this.switcherExpansion();
    const collapsible = tree.length > 1;
    return tree.flatMap((entry) => {
      let portals = entry.portals;
      let expandedByDefault = portals.some((portal) => portal.current);
      if (term) {
        const matchingPortals = portals.filter((portal) => portal.name.toLowerCase().includes(term));
        if (matchingPortals.length > 0) {
          portals = matchingPortals;
          expandedByDefault = true;
        } else if (collapsible && entry.org.name.toLowerCase().includes(term)) {
          expandedByDefault = false;
        } else {
          return [];
        }
      }
      const hasToggle = collapsible && portals.length > 0;
      return [
        {
          ...entry,
          portals,
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

  private readonly createOrgModal = viewChild.required<TemplateRef<unknown>>('createOrgModal');
  private modalRef?: DialogRef;
  // Super admins do not join an organization they create, so they name its admin instead.
  protected readonly createOrgForm = new FormGroup({
    name: new FormControl<string>('', Validators.required),
    adminEmail: new FormControl(
      {value: '', disabled: !this.isSuperAdmin},
      {nonNullable: true, validators: [Validators.required, Validators.email]}
    ),
  });
  protected readonly createOrgFormLoading = signal(false);
  protected readonly createdOrganization = signal<Required<CreateOrganizationResponse> | undefined>(undefined);

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
      this.switchTargetsRefresh$.next();
    }
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

  protected toggleSwitcherEntry(entry: SwitcherEntry) {
    this.switcherExpansion.update((expansion) => new Map(expansion).set(entry.org.id!, !entry.expanded));
  }

  protected async switchOrganization(org: Organization) {
    await this.switchContext({organizationId: org.id!});
  }

  protected async switchPortal(org: Organization, portal: SwitcherPortal) {
    await this.switchContext({
      organizationId: org.id!,
      ...(portal.kind === 'partner' ? {partnerOrganizationId: portal.id} : {customerOrganizationId: portal.id}),
    });
  }

  private async switchContext(request: SwitchContextRequest, targetPath = '/') {
    this.organizationsOpened.set(false);
    try {
      await this.auth.switchContext(request, targetPath);
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
    this.createdOrganization.set(undefined);
  }

  async submitCreateOrgForm() {
    this.createOrgForm.markAllAsTouched();
    if (this.createOrgForm.valid) {
      this.createOrgFormLoading.set(true);
      try {
        const {name, adminEmail} = this.createOrgForm.value;
        const {id, inviteUrl} = await firstValueFrom(this.organizationService.create({name: name!, adminEmail}));
        if (inviteUrl) {
          this.createdOrganization.set({id, inviteUrl});
          this.ctx.reload();
        } else {
          await this.openCreatedOrganization(id);
        }
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.createOrgFormLoading.set(false);
      }
    }
  }

  protected async openCreatedOrganization(organizationId: string) {
    await this.switchContext({organizationId}, '/dashboard?from=new-org');
  }

  async logout() {
    await lastValueFrom(this.auth.logout());
    // This is necessary to flush the caching crud services
    // TODO: implement flushing of services directly and switch to router.navigate(...)
    location.assign('/login');
  }

  protected readonly faAddressBook = faAddressBook;
  protected readonly faArrowLeft = faArrowLeft;
  protected readonly faBuilding = faBuilding;
  protected readonly faCheck = faCheck;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;
  protected readonly faUserCircle = faUserCircle;
}
