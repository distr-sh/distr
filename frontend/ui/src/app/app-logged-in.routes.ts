import {inject} from '@angular/core';
import {CanActivateFn, Router, Routes} from '@angular/router';
import {UserRole} from '@distr-sh/distr-sdk';
import {firstValueFrom, map} from 'rxjs';
import {getRemoteEnvironment} from '../env/remote';
import {AccessTokensComponent} from './access-tokens/access-tokens.component';
import {AdvisoryDetailComponent} from './advisories/advisory-detail.component';
import {AdvisoryListComponent} from './advisories/advisory-list.component';
import {AlertConfigurationsComponent} from './alert-configurations/alert-configurations.component';
import {ApplicationDetailComponent} from './applications/application-detail.component';
import {ApplicationsPageComponent} from './applications/applications-page.component';
import {ArtifactPullsComponent} from './artifacts/artifact-pulls/artifact-pulls.component';
import {ArtifactVersionsComponent} from './artifacts/artifact-versions/artifact-versions.component';
import {ArtifactsComponent} from './artifacts/artifacts/artifacts.component';
import {BillingComponent} from './billing/billing.component';
import {BillingSettingsComponent} from './billing/settings/billing-settings.component';
import {CustomerOrganizationsComponent} from './components/customer-organizations/customer-organizations.component';
import {DashboardComponent} from './components/dashboard/dashboard.component';
import {HomeComponent} from './components/home/home.component';
import {PartnerOrganizationsComponent} from './components/partner-organizations/partner-organizations.component';
import {CustomerUsersComponent} from './components/users/customers/customer-users.component';
import {PartnerUsersComponent} from './components/users/partners/partner-users.component';
import {VendorUsersComponent} from './components/users/vendors/vendor-users.component';
import {CustomerSettingsComponent} from './customer-settings/customer-settings.component';
import {DeploymentTargetDetailComponent} from './deployments/deployment-target-details/deployment-target-detail.component';
import {DeploymentTargetsComponent} from './deployments/deployment-targets.component';
import {CustomerLicenseDetailComponent} from './licenses/customer-license-detail.component';
import {LicenseKeysComponent} from './licenses/license-keys/license-keys.component';
import {LicensesOverviewComponent} from './licenses/licenses-overview.component';
import {NotificationRecordsComponent} from './notification-records/notification-records.component';
import {OrganizationBrandingComponent} from './organization-branding/organization-branding.component';
import {CustomEmailComponent} from './organization-settings/custom-email.component';
import {CustomOidcComponent} from './organization-settings/custom-oidc.component';
import {GeneralSettingsComponent} from './organization-settings/general-settings.component';
import {OrganizationSettingsComponent} from './organization-settings/organization-settings.component';
import {CustomerSecretsPageComponent} from './secrets/customer-secrets-page.component';
import {SecretsPage} from './secrets/secrets-page.component';
import {AuthService} from './services/auth.service';
import {ContextService} from './services/context.service';
import {FeatureFlagService} from './services/feature-flag.service';
import {OrganizationService} from './services/organization.service';
import {ToastService} from './services/toast.service';
import {SidebarLinksPageComponent} from './sidebar-links/sidebar-links-page.component';
import {SubscriptionCallbackComponent} from './subscription/subscription-callback.component';
import {SubscriptionComponent} from './subscription/subscription.component';
import {SupportBundleDetailComponent} from './support-bundles/detail/support-bundle-detail.component';
import {SupportBundleListComponent} from './support-bundles/list/support-bundle-list.component';
import {SupportBundleSettingsComponent} from './support-bundles/vendor/support-bundle-settings.component';
import {AgentsTutorialComponent} from './tutorials/agents/agents-tutorial.component';
import {BrandingTutorialComponent} from './tutorials/branding/branding-tutorial.component';
import {RegistryTutorialComponent} from './tutorials/registry/registry-tutorial.component';
import {TutorialsComponent} from './tutorials/tutorials.component';
import {UsersTutorialComponent} from './tutorials/users/users-tutorial.component';
import {isSubscriptionExpired} from './types/organization';
import {UserGeneralSettingsComponent} from './user-settings/user-general-settings.component';
import {UserIdentitiesComponent} from './user-settings/user-identities.component';
import {UserSecuritySettingsComponent} from './user-settings/user-security-settings.component';
import {UserSettingsComponent} from './user-settings/user-settings.component';

function requiredRoleGuard(...userRole: UserRole[]): CanActivateFn {
  return () => {
    const auth = inject(AuthService);
    if (auth.isSuperAdmin() || auth.hasAnyRole(...userRole)) {
      return true;
    }
    return inject(Router).createUrlTree(['/']);
  };
}

const requireVendor: CanActivateFn = () => {
  if (inject(AuthService).isVendor()) {
    return true;
  }
  return inject(Router).createUrlTree(['/']);
};

const requireCustomer: CanActivateFn = () => {
  if (inject(AuthService).isCustomer()) {
    return true;
  }
  return inject(Router).createUrlTree(['/']);
};

const requireVendorOrPartner: CanActivateFn = () => {
  const auth = inject(AuthService);
  if (auth.isVendor() || auth.isPartner()) {
    return true;
  }
  return inject(Router).createUrlTree(['/']);
};

function licensingEnabledGuard(): CanActivateFn {
  return async () => {
    const featureFlags = inject(FeatureFlagService);
    return await firstValueFrom(featureFlags.isLicensingEnabled$);
  };
}

function notificationsEnabledGuard(): CanActivateFn {
  return async () => {
    const featureFlags = inject(FeatureFlagService);
    return await firstValueFrom(featureFlags.isNotificationsEnabled$);
  };
}

function supportBundlesEnabledGuard(): CanActivateFn {
  return async () => {
    const featureFlags = inject(FeatureFlagService);
    return await firstValueFrom(featureFlags.isSupportBundlesEnabled$);
  };
}

function vendorBillingEnabledGuard(): CanActivateFn {
  return async () => {
    const featureFlags = inject(FeatureFlagService);
    return await firstValueFrom(featureFlags.isVendorBillingEnabled$);
  };
}

// The tab holds the custom domains as well, so either feature makes it reachable.
function customOidcProvidersEnabledGuard(): CanActivateFn {
  return async () => {
    const featureFlags = inject(FeatureFlagService);
    const router = inject(Router);
    return (
      (await firstValueFrom(featureFlags.isCustomOidcProvidersEnabled$)) ||
      (await firstValueFrom(featureFlags.isCustomDomainsEnabled$)) ||
      router.createUrlTree(['/settings/organization/general'])
    );
  };
}

function customerSettingsEnabledGuard(): CanActivateFn {
  return async () => {
    const contextService = inject(ContextService);
    const router = inject(Router);
    const customerOrganization = await firstValueFrom(contextService.getCustomerOrganization());
    return customerOrganization?.features?.includes('oidc_providers') || router.createUrlTree(['/']);
  };
}

function customEmailsEnabledGuard(): CanActivateFn {
  return async () => {
    const featureFlags = inject(FeatureFlagService);
    const router = inject(Router);
    return (
      (await firstValueFrom(featureFlags.isCustomEmailsEnabled$)) ||
      router.createUrlTree(['/settings/organization/general'])
    );
  };
}

function partnerManagementEnabledGuard(): CanActivateFn {
  return async () => {
    const featureFlags = inject(FeatureFlagService);
    return await firstValueFrom(featureFlags.isPartnerManagementEnabled$);
  };
}

function vulnerabilitiesEnabledGuard(): CanActivateFn {
  return async () => {
    const featureFlags = inject(FeatureFlagService);
    return await firstValueFrom(featureFlags.isVulnerabilitiesEnabled$);
  };
}

function registryHostSetOrRedirectGuard(redirectTo: string): CanActivateFn {
  return async () => {
    const router = inject(Router);
    const toast = inject(ToastService);
    const env = await getRemoteEnvironment();
    if ((env.registryHost ?? '').length > 0) {
      return true;
    }
    toast.error('Registry must be enabled first!');
    return router.createUrlTree([redirectTo]);
  };
}

function subscriptionGuard(): CanActivateFn {
  return () => {
    const auth = inject(AuthService);
    const router = inject(Router);
    const organizationService = inject(OrganizationService);
    return (
      auth.isCustomer() ||
      organizationService
        .get()
        .pipe(map((org) => (isSubscriptionExpired(org) ? router.createUrlTree(['/subscription']) : true)))
    );
  };
}

export const routes: Routes = [
  {
    path: '',
    canActivate: [subscriptionGuard()],
    children: [
      {
        path: 'dashboard',
        component: DashboardComponent,
        canActivate: [requireVendorOrPartner],
      },
      {
        path: 'home',
        component: HomeComponent,
        canActivate: [requireCustomer],
      },
      {
        path: 'applications',
        canActivate: [requireVendor],
        children: [
          {
            path: '',
            pathMatch: 'full',
            component: ApplicationsPageComponent,
          },
          {
            path: ':applicationId',
            component: ApplicationDetailComponent,
          },
        ],
      },
      {
        path: 'deployments',
        children: [
          {path: '', pathMatch: 'full', component: DeploymentTargetsComponent},
          {path: ':deploymentTargetId', component: DeploymentTargetDetailComponent},
        ],
      },
      {
        path: 'artifacts',
        children: [
          {path: '', pathMatch: 'full', component: ArtifactsComponent},
          {path: ':id', component: ArtifactVersionsComponent},
        ],
      },
      {
        path: 'artifact-pulls',
        component: ArtifactPullsComponent,
        canActivate: [requireVendorOrPartner],
      },
      {
        path: 'customers',
        component: CustomerOrganizationsComponent,
        canActivate: [requireVendorOrPartner],
      },
      {
        path: 'customers/:customerOrganizationId',
        canActivate: [requireVendorOrPartner],
        children: [
          {path: 'users', component: CustomerUsersComponent},
          {path: 'secrets', component: CustomerSecretsPageComponent},
          {path: 'links', component: SidebarLinksPageComponent},
          {
            path: 'settings',
            component: CustomerSettingsComponent,
            canActivate: [customOidcProvidersEnabledGuard()],
          },
          {path: '', pathMatch: 'full', redirectTo: 'users'},
        ],
      },
      {
        path: 'partners',
        component: PartnerOrganizationsComponent,
        canActivate: [requireVendor, partnerManagementEnabledGuard()],
      },
      {
        path: 'partners/:partnerOrganizationId',
        canActivate: [requireVendor, partnerManagementEnabledGuard()],
        children: [
          {path: 'users', component: PartnerUsersComponent},
          {path: '', pathMatch: 'full', redirectTo: 'users'},
        ],
      },
      {
        path: 'users',
        component: VendorUsersComponent,
        canActivate: [requiredRoleGuard('admin')],
      },
      {
        path: 'secrets',
        component: SecretsPage,
      },
      {
        path: 'license-keys',
        component: LicenseKeysComponent,
        canActivate: [requireCustomer, licensingEnabledGuard()],
      },
      {
        path: 'branding',
        component: OrganizationBrandingComponent,
        data: {userRole: 'vendor'},
        canActivate: [requireVendor, requiredRoleGuard('read_write', 'admin')],
      },
      {
        path: 'billing',
        canActivate: [requireVendor, vendorBillingEnabledGuard()],
        children: [
          {
            path: '',
            pathMatch: 'full',
            component: BillingComponent,
          },
          {
            path: 'settings',
            component: BillingSettingsComponent,
            canActivate: [requiredRoleGuard('admin')],
          },
        ],
      },
      {
        path: 'licenses',
        canActivate: [requireVendorOrPartner, licensingEnabledGuard()],
        data: {userRole: 'vendor'},
        children: [
          {
            path: '',
            pathMatch: 'full',
            component: LicensesOverviewComponent,
          },
          {
            path: ':customerOrganizationId',
            component: CustomerLicenseDetailComponent,
          },
        ],
      },
      {
        path: 'settings',
        children: [
          {
            path: '',
            pathMatch: 'full',
            redirectTo: 'organization',
          },
          {
            path: 'customer',
            component: CustomerSettingsComponent,
            canActivate: [requireCustomer, requiredRoleGuard('admin'), customerSettingsEnabledGuard()],
          },
          {
            path: 'organization',
            component: OrganizationSettingsComponent,
            data: {userRole: 'vendor'},
            canActivate: [requireVendor, requiredRoleGuard('admin')],
            children: [
              {
                path: '',
                pathMatch: 'full',
                redirectTo: 'general',
              },
              {
                path: 'general',
                component: GeneralSettingsComponent,
              },
              {
                path: 'identity-provider',
                component: CustomOidcComponent,
                canActivate: [customOidcProvidersEnabledGuard()],
              },
              {
                path: 'email',
                component: CustomEmailComponent,
                canActivate: [customEmailsEnabledGuard()],
              },
            ],
          },
          {
            path: 'profile',
            component: UserSettingsComponent,
            children: [
              {
                path: '',
                pathMatch: 'full',
                redirectTo: 'general',
              },
              {
                path: 'general',
                component: UserGeneralSettingsComponent,
              },
              {
                path: 'security',
                component: UserSecuritySettingsComponent,
              },
              {
                path: 'identities',
                component: UserIdentitiesComponent,
              },
            ],
          },
          {
            path: 'access-tokens',
            component: AccessTokensComponent,
          },
        ],
      },
      {
        path: 'tutorials',
        canActivate: [requireVendor, requiredRoleGuard('admin')],
        children: [
          {
            path: '',
            pathMatch: 'full',
            component: TutorialsComponent,
          },
          {
            path: 'users',
            component: UsersTutorialComponent,
          },
          {
            path: 'agents',
            component: AgentsTutorialComponent,
          },
          {
            path: 'branding',
            component: BrandingTutorialComponent,
          },
          {
            path: 'registry',
            canActivate: [registryHostSetOrRedirectGuard('/tutorials')],
            component: RegistryTutorialComponent,
          },
        ],
      },
      {
        path: 'notifications',
        canActivate: [notificationsEnabledGuard()],
        children: [
          {
            path: 'alert-configurations',
            component: AlertConfigurationsComponent,
          },
          {
            path: 'history',
            component: NotificationRecordsComponent,
          },
        ],
      },
      {
        path: 'support-bundles',
        canActivate: [requireVendorOrPartner, supportBundlesEnabledGuard()],
        children: [
          {
            path: '',
            pathMatch: 'full',
            component: SupportBundleListComponent,
          },
          {
            path: 'settings',
            component: SupportBundleSettingsComponent,
            canActivate: [requireVendor, requiredRoleGuard('read_write', 'admin')],
          },
          {
            path: ':supportBundleId',
            component: SupportBundleDetailComponent,
          },
        ],
      },
      {
        path: 'support',
        canActivate: [requireCustomer],
        children: [
          {
            path: '',
            pathMatch: 'full',
            component: SupportBundleListComponent,
          },
          {
            path: ':supportBundleId',
            component: SupportBundleDetailComponent,
          },
        ],
      },
      {
        path: 'advisories',
        canActivate: [requireVendorOrPartner, vulnerabilitiesEnabledGuard()],
        children: [
          {
            path: '',
            pathMatch: 'full',
            component: AdvisoryListComponent,
          },
          {
            path: ':advisoryId',
            component: AdvisoryDetailComponent,
          },
        ],
      },
      {
        path: 'security',
        canActivate: [requireCustomer, vulnerabilitiesEnabledGuard()],
        children: [
          {
            path: '',
            pathMatch: 'full',
            component: AdvisoryListComponent,
          },
          {
            path: ':advisoryId',
            component: AdvisoryDetailComponent,
          },
        ],
      },
    ],
  },
  {
    path: 'subscription',
    canActivate: [requireVendor],
    children: [
      {
        path: '',
        pathMatch: 'full',
        component: SubscriptionComponent,
      },
      {
        path: 'callback',
        component: SubscriptionCallbackComponent,
      },
    ],
  },
];
