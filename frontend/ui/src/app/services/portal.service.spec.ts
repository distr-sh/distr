import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {Portal} from '../types/portal';
import {PortalService} from './portal.service';

const portalResponse: Portal = {
  customDomain: false,
  loginConfig: {
    registrationEnabled: true,
    oidcGithubEnabled: true,
    oidcGoogleEnabled: true,
    oidcMicrosoftEnabled: false,
    oidcGenericEnabled: false,
  },
};

describe('PortalService', () => {
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({providers: [provideHttpClient(), provideHttpClientTesting()]});
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpTesting.verify());

  it('exposes the login methods offered on this host', () => {
    const service = TestBed.inject(PortalService);
    httpTesting.expectOne('/api/public/v1/portal').flush(portalResponse);

    expect(service.loginConfig()).toEqual(portalResponse.loginConfig);
  });

  it('offers no login method when the response has no body', () => {
    const service = TestBed.inject(PortalService);
    httpTesting.expectOne('/api/public/v1/portal').flush(null, {status: 204, statusText: 'No Content'});

    expect(service.loginConfig().registrationEnabled).toBe(false);
    expect(service.loginConfig().oidcGithubEnabled).toBe(false);
  });

  it('requests the portal configuration once and replays it to every consumer', () => {
    const service = TestBed.inject(PortalService);
    httpTesting.expectOne('/api/public/v1/portal').flush({...portalResponse, customDomain: true});

    let customDomain = false;
    service.portal$.subscribe((portal) => (customDomain = portal.customDomain));

    expect(customDomain).toBe(true);
  });
});
