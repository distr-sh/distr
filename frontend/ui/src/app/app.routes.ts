import {inject} from '@angular/core';
import {
  ActivatedRouteSnapshot,
  CanActivateFn,
  createUrlTreeFromSnapshot,
  Router,
  RouterStateSnapshot,
  Routes,
} from '@angular/router';
import {firstValueFrom} from 'rxjs';
import {ForgotComponent} from './forgot/forgot.component';
import {InviteComponent} from './invite/invite.component';
import {LoginComponent} from './login/login.component';
import {MaintenanceComponent} from './maintenance/maintenance.component';
import {PasswordResetComponent} from './password-reset/password-reset.component';
import {RegisterComponent} from './register/register.component';
import {actionFlowPath, AuthService} from './services/auth.service';
import {ToastService} from './services/toast.service';
import {VerifyComponent} from './verify/verify.component';

const emailVerificationGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const toast = inject(ToastService);
  const router = inject(Router);
  const claims = auth.getClaims();
  if (claims?.email_verified) {
    await firstValueFrom(auth.confirmEmailVerification());
    toast.success('Your email has been verified');
    await firstValueFrom(auth.logout());
    return router.createUrlTree(['/login'], {queryParams: {email: claims.email}});
  }
  return true;
};

const jwtParamRedirectGuard: CanActivateFn = (route: ActivatedRouteSnapshot) => {
  const auth = inject(AuthService);
  const jwt = route.queryParamMap.get('jwt');
  if (jwt === null) {
    return true;
  } else {
    // TODO: flush crud service caches
    auth.actionToken = jwt;
    const newtree = createUrlTreeFromSnapshot(route, [], null, null);
    delete newtree.queryParams['jwt']; // prevent infinite loop
    return newtree;
  }
};

const jwtAuthGuard: CanActivateFn = (_: ActivatedRouteSnapshot, state: RouterStateSnapshot) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const claims = auth.getClaims();
  if (!claims) {
    return router.createUrlTree(['/login']);
  }
  if (!auth.isLoggedIn()) {
    // The token of a credential-setup flow is not a session: it carries no organization and is rejected by every
    // organization-scoped endpoint. Confine it to the page of its own flow, so it can neither reach the app nor
    // bounce between the redirects that decide where a session belongs.
    const path = actionFlowPath(claims);
    return state.url.startsWith(path) ? true : router.createUrlTree([path]);
  }
  if (!claims.email_verified) {
    return state.url === '/verify' ? true : router.createUrlTree(['/verify']);
  }
  return true;
};

const inviteComponentGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  try {
    const {active} = await firstValueFrom(auth.getUserStatus());
    if (!active) {
      return true;
    }
  } catch (e) {}
  auth.actionToken = null;
  return router.createUrlTree(['/login']);
};

const baseRouteRedirectGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  if (auth.isVendor() || auth.isPartner()) {
    return router.createUrlTree(['/dashboard']);
  } else {
    return router.createUrlTree(['/home']);
  }
};

export const routes: Routes = [
  {path: 'maintenance', component: MaintenanceComponent},
  {path: 'login', component: LoginComponent},
  {path: 'register', component: RegisterComponent},
  {path: 'forgot', component: ForgotComponent},
  {
    path: '',
    canActivate: [jwtParamRedirectGuard, jwtAuthGuard],
    children: [
      {
        path: '',
        pathMatch: 'full',
        canActivate: [baseRouteRedirectGuard],
        children: [],
      },
      {
        path: 'verify',
        component: VerifyComponent,
        canActivate: [emailVerificationGuard],
      },
      {path: 'reset', component: PasswordResetComponent},
      {path: 'join', component: InviteComponent, canActivate: [inviteComponentGuard]},
      {
        path: '',
        loadComponent: () => import('./components/nav-shell.component').then((m) => m.NavShellComponent),
        loadChildren: () => import('./app-logged-in.routes').then((m) => m.routes),
      },
    ],
  },
  {
    path: '**',
    redirectTo: '/',
  },
];
