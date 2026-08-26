import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {provideRouter, Router} from '@angular/router';
import {Portal, RegistrationMode} from '../types/portal';
import {RegisterComponent} from './register.component';

function portal(registration: RegistrationMode): Portal {
  return {
    customDomain: false,
    loginConfig: {
      registration,
      oidcGithubEnabled: false,
      oidcGoogleEnabled: false,
      oidcMicrosoftEnabled: false,
      oidcGenericEnabled: false,
      oidcProviders: [],
    },
  };
}

describe('RegisterComponent', () => {
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpTesting.verify());

  function render(registration: RegistrationMode) {
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    TestBed.createComponent(RegisterComponent).detectChanges();
    httpTesting.expectOne('/api/public/v1/portal').flush(portal(registration));
    return navigate;
  }

  it('stays on the registration page when registration is hidden', () => {
    expect(render('hidden')).not.toHaveBeenCalled();
  });

  it('redirects to the login page when registration is disabled', () => {
    expect(render('disabled')).toHaveBeenCalledWith(['/login'], {replaceUrl: true});
  });
});
