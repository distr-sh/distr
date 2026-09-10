import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {NavigationBehaviorOptions, provideRouter, Router} from '@angular/router';
import {RouterTestingHarness} from '@angular/router/testing';
import {AccessToken} from '@distr-sh/distr-sdk';
import {of} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {OverlayService} from '../services/overlay.service';
import {AccessTokenDetailComponent} from './access-token-detail.component';

const legacyToken: AccessToken = {
  id: '11111111-1111-1111-1111-111111111111',
  keyId: 'distr-2LTMfjV5xU8sJfF1M0hIm8',
  createdAt: '2026-01-01T00:00:00Z',
  label: 'legacy',
  secrets: [],
};

describe('AccessTokenDetailComponent', () => {
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([{path: 'settings/access-tokens/:accessTokenId', component: AccessTokenDetailComponent}]),
        {provide: AuthService, useValue: {getClaims: () => ({role: 'admin'})}},
        {provide: OverlayService, useValue: {confirm: () => of(false)}},
      ],
    });
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpTesting.verify());

  async function open(token: AccessToken, extras?: NavigationBehaviorOptions) {
    const harness = await RouterTestingHarness.create();
    await TestBed.inject(Router).navigateByUrl(`/settings/access-tokens/${token.id}`, extras);
    harness.detectChanges();
    httpTesting.expectOne('/api/v1/settings/tokens').flush([token]);
    await harness.fixture.whenStable();
    harness.detectChanges();
    return harness.routeNativeElement!.textContent ?? '';
  }

  it('opens a token that has neither secrets nor an explicit role', async () => {
    expect(await open(legacyToken)).toContain('Secure token');
  });

  it('shows the token a create navigation handed over', async () => {
    const state = {createdToken: {...legacyToken, key: 'distr-2LTMfjV5xU8sJfF1M0hIm8_secret'}};
    expect(await open(legacyToken, {state})).toContain(state.createdToken.key);
  });
});
