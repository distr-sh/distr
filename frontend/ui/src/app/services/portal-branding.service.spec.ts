import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {Portal} from '../types/portal';
import {AuthService, JWTClaims} from './auth.service';
import {PortalBrandingService} from './portal-branding.service';

// Regression test: requesting the organization branding with a password reset or invite token (neither of which
// carries an organization) is answered with 401, which makes the token interceptor discard the token and send the
// user to the "your link has expired" page in the middle of the flow.

const portalResponse: Portal = {
  customDomain: false,
  loginConfig: {
    registrationEnabled: true,
    oidcGithubEnabled: false,
    oidcGoogleEnabled: false,
    oidcMicrosoftEnabled: false,
    oidcGenericEnabled: false,
  },
};

describe('PortalBrandingService', () => {
  let httpTesting: HttpTestingController;
  let claims: Partial<JWTClaims> | undefined;

  beforeEach(() => {
    claims = undefined;
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {provide: AuthService, useValue: {isLoggedIn: () => claims?.org !== undefined}},
      ],
    });
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpTesting.verify());

  it('does not request the organization branding with a password reset token', () => {
    claims = {scope: 'password_reset', email: 'test@example.com'};

    TestBed.inject(PortalBrandingService).apply();
    httpTesting.expectOne('/api/public/v1/portal').flush(portalResponse);

    httpTesting.expectNone('/api/v1/organization/branding');
  });

  it('requests the organization branding when logged in with an organization', () => {
    claims = {org: '2f4b9b3a-2f2e-4a3e-8f0a-6b0f5e2f7a11', email: 'test@example.com'};

    TestBed.inject(PortalBrandingService).apply();
    httpTesting.expectOne('/api/public/v1/portal').flush(portalResponse);

    httpTesting.expectOne('/api/v1/organization/branding').flush({});
  });
});
