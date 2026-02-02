import {inject} from '@angular/core';
import {CanActivateFn, Router, Routes} from '@angular/router';
import {UserRole} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {firstValueFrom, map} from 'rxjs';
import {getRemoteEnvironment} from '../env/remote';
import {AccessTokensComponent} from './access-tokens/access-tokens.component';
import {ApplicationDetailComponent} from './applications/application-detail.component';
import {ApplicationsPageComponent} from './applications/applications-page.component';
import {ArtifactLicensesComponent} from './artifacts/artifact-licenses/artifact-licenses.component';
import {ArtifactPullsComponent} from './artifacts/artifact-pulls/artifact-pulls.component';
import {ArtifactVersionsComponent} from './artifacts/artifact-versions/artifact-versions.component';
import {ArtifactsComponent} from './artifacts/artifacts/artifacts.component';
import {CustomerOrganizationsComponent} from './components/customer-organizations/customer-organizations.component';
import {DashboardComponent} from './components/dashboard/dashboard.component';
import {HomeComponent} from './components/home/home.component';
import {CustomerUsersComponent} from './components/users/customers/customer-users.component';
import {VendorUsersComponent} from './components/users/vendors/vendor-users.component';
import {DeploymentStatusNotificationConfigurationsComponent} from './deployment-status-notification-configurations/deployment-status-notification-configurations.component';
import {DeploymentTargetsComponent} from './deployments/deployment-targets.component';
import {LicensesComponent} from './licenses/licenses.component';
import {NotificationRecordsComponent} from './notification-records/notification-records.component';
import {OrganizationBrandingComponent} from './organization-branding/organization-branding.component';
import {OrganizationSettingsComponent} from './organization-settings/organization-settings.component';
import {CustomerSecretsPageComponent} from './secrets/customer-secrets-page.component';
import {SecretsPage} from './secrets/secrets-page.component';
import {AuthService} from './services/auth.service';
import {FeatureFlagService} from './services/feature-flag.service';
import {OrganizationService} from './services/organization.service';
import {ToastService} from './services/toast.service';
import {SubscriptionCallbackComponent} from './subscription/subscription-callback.component';
import {SubscriptionComponent} from './subscription/subscription.component';
import {AgentsTutorialComponent} from './tutorials/agents/agents-tutorial.component';
import {BrandingTutorialComponent} from './tutorials/branding/branding-tutorial.component';
import {RegistryTutorialComponent} from './tutorials/registry/registry-tutorial.component';
import {TutorialsComponent} from './tutorials/tutorials.component';
import {UserSettingsComponent} from './user-settings/user-settings.component';

function requiredRoleGuard(...userRole: UserRole[]): CanActivateFn {
  return () => {
    if (inject(AuthService).hasAnyRole(...userRole)) {
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

function licensingEnabledGuard(): CanActivateFn {
  return async () => {
    const featureFlags = inject(FeatureFlagService);
    return await firstValueFrom(featureFlags.isLicensingEnabled$);
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
        .pipe(
          map((org) =>
            org.subscriptionType !== 'community' && dayjs(org.subscriptionEndsAt).isBefore()
              ? router.createUrlTree(['/subscription'])
              : true
          )
        )
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
        canActivate: [requireVendor],
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
      {path: 'deployments', component: DeploymentTargetsComponent},
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
        canActivate: [requireVendor],
      },
      {
        path: 'customers',
        component: CustomerOrganizationsComponent,
        canActivate: [requireVendor],
      },
      {
        path: 'customers/:customerOrganizationId',
        canActivate: [requireVendor],
        children: [
          {path: 'users', component: CustomerUsersComponent},
          {path: 'secrets', component: CustomerSecretsPageComponent},
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
        path: 'branding',
        component: OrganizationBrandingComponent,
        data: {userRole: 'vendor'},
        canActivate: [requireVendor, requiredRoleGuard('read_write', 'admin')],
      },
      {
        path: 'licenses',
        canActivate: [requireVendor, licensingEnabledGuard()],
        data: {userRole: 'vendor'},
        children: [
          {
            path: 'applications',
            component: LicensesComponent,
          },
          {
            path: 'artifacts',
            component: ArtifactLicensesComponent,
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
            path: 'organization',
            component: OrganizationSettingsComponent,
            data: {userRole: 'vendor'},
            canActivate: [requireVendor, requiredRoleGuard('admin')],
          },
          {
            path: 'profile',
            component: UserSettingsComponent,
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
        children: [
          {
            path: 'deployment-status',
            component: DeploymentStatusNotificationConfigurationsComponent,
          },
          {
            path: 'history',
            component: NotificationRecordsComponent,
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
