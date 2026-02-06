import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe, TitleCasePipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, computed, inject, input, OnInit, TemplateRef, ViewChild} from '@angular/core';
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
import dayjs from 'dayjs';
import {catchError, EMPTY, lastValueFrom, map, of} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {SecureImagePipe} from '../../../util/secureImage';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {RequireCustomerDirective, RequireVendorDirective} from '../../directives/required-role.directive';
import {AuthService} from '../../services/auth.service';
import {ContextService} from '../../services/context.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SidebarService} from '../../services/sidebar.service';
import {ToastService} from '../../services/toast.service';
import {UsersService} from '../../services/users.service';
import {Organization, OrganizationWithUserRole} from '../../types/organization';
import {ColorSchemeSwitcherComponent} from '../color-scheme-switcher/color-scheme-switcher.component';
import {NavBarSubscriptionBannerComponent} from './nav-bar-subscription-banner/nav-bar-subscription-banner.component';

type SwitchOptions = {
  currentOrg: Organization;
  availableOrgs: OrganizationWithUserRole[];
  isVendorSomewhere: boolean;
};

@Component({
  selector: 'app-nav-bar',
  standalone: true,
  templateUrl: './nav-bar.component.html',
  imports: [
    ColorSchemeSwitcherComponent,
    NavBarSubscriptionBannerComponent,
    OverlayModule,
    FaIconComponent,
    RouterLink,
    SecureImagePipe,
    AsyncPipe,
    TitleCasePipe,
    AutotrimDirective,
    ReactiveFormsModule,
    RequireVendorDirective,
    RequireCustomerDirective,
  ],
  animations: [dropdownAnimation, modalFlyInOut],
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
    return this.allOrgs().filter((org) => org.id !== current?.id);
  });
  protected readonly currentOrg = toSignal(this.ctx.getOrganization());
  protected readonly isVendorSomewhere = computed(() =>
    this.allOrgs()
      .filter((org) => org !== undefined)
      .some((org) => org.customerOrganizationId === undefined)
  );
  protected readonly isTrial = computed(() => this.currentOrg()?.subscriptionType === 'trial');
  protected readonly isSubscriptionExpired = computed(() => {
    const org = this.currentOrg();
    if (org && org.subscriptionType !== 'community') {
      return dayjs(org.subscriptionEndsAt).isBefore();
    }
    return false;
  });

  userOpened = false;
  organizationsOpened = false;
  logoUrl = '/distr-logo.svg';
  customerSubtitle = 'Customer Portal';

  protected readonly faBarsStaggered = faBarsStaggered;
  protected readonly tutorial = toSignal(this.route.queryParams.pipe(map((params) => params['tutorial'])));

  public readonly isSubscriptionBannerVisible = input<boolean>();
  public readonly isSidebarVisible = input<boolean>();

  @ViewChild('createOrgModal') private createOrgModal!: TemplateRef<unknown>;
  private modalRef?: DialogRef;
  protected readonly createOrgForm = new FormGroup({
    name: new FormControl<string>('', Validators.required),
  });

  public async ngOnInit() {
    try {
      await this.initBranding();
    } catch (e) {
      console.error(e);
    }
  }

  private async initBranding() {
    if (this.auth.isCustomer()) {
      try {
        const branding = await lastValueFrom(this.organizationBranding.get());
        if (branding.logo) {
          this.logoUrl = `data:${branding.logoContentType};base64,${branding.logo}`;
        }
        if (branding.title) {
          this.customerSubtitle = branding.title;
        }
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg && e instanceof HttpErrorResponse && e.status !== 404) {
          this.toast.error(msg);
        }
      }
    }
  }

  async switchContext(org: Organization, targetPath = '/') {
    this.organizationsOpened = false;
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
    this.modalRef = this.overlay.showModal(this.createOrgModal);
    this.modalRef.addOnClosedHook((_) => {
      this.organizationsOpened = false;
    });
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
