import {CdkConnectedOverlay, CdkOverlayOrigin} from '@angular/cdk/overlay';
import {NgTemplateOutlet} from '@angular/common';
import {Component, inject, input, signal, WritableSignal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink, RouterLinkActive} from '@angular/router';
import {CustomerOrganization} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faAddressBook,
  faArrowRightLong,
  faAsterisk,
  faBox,
  faBoxesStacked,
  faBullhorn,
  faChevronDown,
  faCreditCard,
  faDashboard,
  faGear,
  faHome,
  faKey,
  faLightbulb,
  faPalette,
  faUsers,
} from '@fortawesome/free-solid-svg-icons';
import {map} from 'rxjs';
import {buildConfig} from '../../../buildconfig';
import {environment} from '../../../env/env';
import {RequireCustomerDirective, RequireVendorDirective} from '../../directives/required-role.directive';
import {AuthService} from '../../services/auth.service';
import {ContextService} from '../../services/context.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {OrganizationService} from '../../services/organization.service';
import {SidebarService} from '../../services/sidebar.service';
import {TutorialsService} from '../../services/tutorials.service';

@Component({
  selector: 'app-side-bar',
  standalone: true,
  templateUrl: './side-bar.component.html',
  imports: [
    RouterLink,
    FaIconComponent,
    RouterLinkActive,
    CdkOverlayOrigin,
    CdkConnectedOverlay,
    NgTemplateOutlet,
    RequireVendorDirective,
    RequireCustomerDirective,
  ],
})
export class SideBarComponent {
  protected readonly auth = inject(AuthService);
  protected readonly sidebar = inject(SidebarService);
  private readonly organizationService = inject(OrganizationService);
  private readonly tutorialsService = inject(TutorialsService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly contextService = inject(ContextService);

  protected readonly buildConfig = buildConfig;
  protected readonly edition = environment.edition;

  protected readonly faDashboard = faDashboard;
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faLightbulb = faLightbulb;
  protected readonly faKey = faKey;
  protected readonly faGear = faGear;
  protected readonly faUsers = faUsers;
  protected readonly faPalette = faPalette;
  protected readonly faAddressBook = faAddressBook;
  protected readonly faBox = faBox;
  protected readonly faCreditCard = faCreditCard;
  protected readonly faArrowRightLong = faArrowRightLong;
  protected readonly faHome = faHome;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faAsterisk = faAsterisk;
  protected readonly faBullhorn = faBullhorn;

  protected feedbackAlert = true;
  protected readonly agentsSubMenuOpen = signal(true);
  protected readonly licenseSubMenuOpen = signal(false);
  protected readonly registrySubMenuOpen = signal(true);
  protected readonly notificationsSubMenuOpen = signal(false);
  protected readonly licenseOverlayOpen = signal(false);
  protected readonly notificationsOverlayOpen = signal(false);

  protected readonly isAllTutorialsStarted = toSignal(this.tutorialsService.allStarted$);
  protected readonly isLicensingFeatureEnabled = toSignal(this.featureFlags.isLicensingEnabled$);
  protected readonly isNotificationsFeatureEnabled = toSignal(this.featureFlags.isNotificationsEnabled$);

  public readonly isSubscriptionBannerVisible = input<boolean>();
  public readonly isSidebarVisible = input<boolean>();

  protected readonly hasNoSubscription = toSignal(
    this.organizationService
      .get()
      .pipe(
        map(
          (org) =>
            !(
              org.subscriptionType === 'starter' ||
              org.subscriptionType === 'pro' ||
              org.subscriptionType === 'enterprise'
            )
        )
      ),
    {initialValue: false}
  );

  protected readonly customerOrgFeatures = toSignal(
    this.contextService.getCustomerOrganization().pipe(
      map((customerOrg: CustomerOrganization | undefined): string[] => {
        return customerOrg?.features || [];
      })
    ),
    {initialValue: [] as string[]}
  );

  protected hasCustomerOrganizationFeature(feature: string): boolean {
    const features = this.customerOrgFeatures();
    return Array.isArray(features) && features.includes(feature);
  }

  protected toggle(signal: WritableSignal<boolean>) {
    signal.update((val) => !val);
  }
}
