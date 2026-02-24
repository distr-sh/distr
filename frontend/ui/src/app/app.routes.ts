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
import {PasswordResetComponent} from './password-reset/password-reset.component';
import {RegisterComponent} from './register/register.component';
import {AuthService} from './services/auth.service';
import {SettingsService} from './services/settings.service';
import {ToastService} from './services/toast.service';
import {UsersService} from './services/users.service';
import {VerifyComponent} from './verify/verify.component';

const emailVerificationGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const settings = inject(SettingsService);
  const toast = inject(ToastService);
  const router = inject(Router);
  const claims = auth.getClaims();
  if (claims?.email_verified) {
    await firstValueFrom(settings.confirmEmailVerification());
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
  if (claims) {
    if (claims.password_reset) {
      if (state.url === '/reset') {
        return true;
      } else {
        return router.createUrlTree(['/reset']);
      }
    } else if (!claims.email_verified) {
      if (state.url === '/verify') {
        return true;
      } else {
        return router.createUrlTree(['/verify']);
      }
    } else {
      return true;
    }
  } else {
    return router.createUrlTree(['/login']);
  }
};

const inviteComponentGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const users = inject(UsersService);
  const router = inject(Router);
  try {
    const {active} = await firstValueFrom(users.getUserStatus());
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
  if (auth.isVendor()) {
    return router.createUrlTree(['/dashboard']);
  } else {
    return router.createUrlTree(['/home']);
  }
};

export const routes: Routes = [
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
