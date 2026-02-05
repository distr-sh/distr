import {HttpClient, HttpErrorResponse, HttpInterceptorFn, HttpRequest} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {LoginResponse, TokenResponse, UserRole} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {jwtDecode} from 'jwt-decode';
import {catchError, map, Observable, of, tap, throwError} from 'rxjs';
import {Organization} from '../types/organization';

const tokenStorageKey = 'cloud_token';
const actionTokenStorageKey = 'distr_action_token';
const authBaseUrl = '/api/v1/auth';

export interface JWTClaims {
  sub: string;
  org: string;
  c_org: string;
  email: string;
  password_reset: boolean;
  email_verified: boolean;
  name: string;
  exp: string;
  role: UserRole;
  image_url: string | undefined;
  is_super_admin?: boolean;
  [claim: string]: unknown;
}

export interface LoginConfig {
  registrationEnabled?: boolean;
  oidcGithubEnabled?: boolean;
  oidcGoogleEnabled?: boolean;
  oidcMicrosoftEnabled?: boolean;
  oidcGenericEnabled?: boolean;
}

@Injectable({providedIn: 'root'})
export class AuthService {
  private readonly httpClient = inject(HttpClient);

  private get token(): string | null {
    return localStorage.getItem(tokenStorageKey);
  }

  private set token(value: string | null) {
    if (value !== null) {
      localStorage.setItem(tokenStorageKey, value);
    } else {
      localStorage.removeItem(tokenStorageKey);
    }
  }

  public get actionToken(): string | null {
    return sessionStorage.getItem(actionTokenStorageKey);
  }

  public set actionToken(value: string | null) {
    if (value !== null) {
      sessionStorage.setItem(actionTokenStorageKey, value);
    } else {
      sessionStorage.removeItem(actionTokenStorageKey);
    }
  }

  public hasRole(role: UserRole): boolean {
    return this.getClaims()?.role === role;
  }

  public hasAnyRole(...roles: UserRole[]): boolean {
    return roles.some((role) => this.hasRole(role));
  }

  public isVendor(): boolean {
    return this.getClaims()?.c_org === undefined;
  }

  public isCustomer(): boolean {
    return this.getClaims()?.c_org !== undefined;
  }

  public isSuperAdmin(): boolean {
    return this.getClaims()?.is_super_admin === true;
  }

  public login(email: string, password: string, mfaCode?: string): Observable<{requiresMfa: boolean}> {
    return this.httpClient.post<LoginResponse>(`${authBaseUrl}/login`, {email, password, mfaCode}).pipe(
      tap((r) => {
        if (!r.requiresMfa) {
          this.token = r.token;
          this.actionToken = null;
        }
      }),
      map((r) => ({requiresMfa: r.requiresMfa}))
    );
  }

  public loginWithToken(jwt: string) {
    this.token = jwt;
    this.actionToken = null;
  }

  public resetPassword(email: string): Observable<void> {
    return this.httpClient.post<void>(`${authBaseUrl}/reset`, {email});
  }

  public loginConfig(): Observable<LoginConfig> {
    return this.httpClient.get<LoginConfig>(`${authBaseUrl}/login/config`).pipe(catchError(() => of({})));
  }

  public register(email: string, name: string | null | undefined, password: string): Observable<void> {
    let body: any = {email, password};
    if (name) {
      body = {...body, name};
    }
    return this.httpClient.post<void>(`${authBaseUrl}/register`, body);
  }

  public getClaims(): JWTClaims | undefined {
    const {claims} = this.getTokenAndClaims();
    return claims;
  }

  public getTokenAndClaims(): {token: string | null; claims: JWTClaims | undefined} {
    const actionToken = this.actionToken;
    if (actionToken !== null) {
      try {
        return {token: actionToken, claims: jwtDecode(actionToken)};
      } catch (e) {
        console.error(e);
      }
    } else {
      const token = this.token;
      if (token !== null) {
        try {
          return {token, claims: jwtDecode(token)};
        } catch (e) {
          console.error(e);
        }
      }
    }
    return {token: null, claims: undefined};
  }

  public switchContext(org: Organization): Observable<boolean> {
    return this.httpClient
      .post<TokenResponse | undefined>(`${authBaseUrl}/switch-context`, {organizationId: org.id})
      .pipe(
        map((r) => {
          if (r) {
            this.token = r.token;
            this.actionToken = null;
            return true;
          }
          return false;
        })
      );
  }

  public logout(): Observable<void> {
    this.token = null;
    this.actionToken = null;
    return of(undefined);
  }
}

export const tokenInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  if (authenticatedRoute(req)) {
    const {token, claims} = auth.getTokenAndClaims();
    try {
      if (claims && dayjs.unix(parseInt(claims.exp)).isAfter(dayjs())) {
        return next(req.clone({headers: req.headers.set('Authorization', `Bearer ${token}`)})).pipe(
          tap({
            error: (e) => {
              if (e instanceof HttpErrorResponse && e.status == 401) {
                auth.logout();
                removeJwtQueryParamAndRefresh(claims?.email);
              }
            },
          })
        );
      } else {
        auth.logout();
        removeJwtQueryParamAndRefresh(claims?.email);
        return throwError(() => new Error('no token or token has expired'));
      }
    } catch (cause) {
      return throwError(() => new Error('no token', {cause}));
    }
  } else {
    return next(req);
  }
};

function authenticatedRoute(req: HttpRequest<unknown>): boolean {
  return !req.url.startsWith(authBaseUrl) || req.url === `${authBaseUrl}/switch-context`;
}

function removeJwtQueryParamAndRefresh(email?: string) {
  const url = new URL(location.href);
  if (url.searchParams.has('jwt')) {
    url.searchParams.delete('jwt');
  }
  if (url.pathname === '/join') {
    url.pathname = '/forgot';
    url.searchParams.append('reason', 'invite-expired');
  } else if (url.pathname === '/reset') {
    url.pathname = '/forgot';
    url.searchParams.append('reason', 'reset-expired');
  } else {
    url.pathname = '/login';
    url.searchParams.append('reason', 'session-expired');
  }
  if (email) {
    url.searchParams.append('email', email);
  }
  location.assign(url);
}
