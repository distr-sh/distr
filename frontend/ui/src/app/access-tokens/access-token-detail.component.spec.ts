import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {provideRouter, Router} from '@angular/router';
import {RouterTestingHarness} from '@angular/router/testing';
import {AccessToken, AccessTokenWithKey} from '@distr-sh/distr-sdk';
import {of} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {CreatedAccessTokenStore} from '../services/created-access-token.service';
import {OverlayService} from '../services/overlay.service';
import {AccessTokenDetailComponent} from './access-token-detail.component';

const legacyToken: AccessToken = {
  id: '11111111-1111-1111-1111-111111111111',
  keyId: 'distr-2LTMfjV5xU8sJfF1M0hIm8',
  createdAt: '2026-01-01T00:00:00Z',
  label: 'legacy',
  secrets: [],
};

const sibling: AccessToken = {
  ...legacyToken,
  id: '22222222-2222-2222-2222-222222222222',
  label: 'sibling',
};

const created: AccessTokenWithKey = {...legacyToken, key: 'distr-2LTMfjV5xU8sJfF1M0hIm8_secret'};

describe('AccessTokenDetailComponent', () => {
  let httpTesting: HttpTestingController;
  let harness: RouterTestingHarness;

  beforeEach(async () => {
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
    harness = await RouterTestingHarness.create();
  });

  afterEach(() => httpTesting.verify());

  function rendered() {
    harness.detectChanges();
    return harness.routeNativeElement?.textContent ?? '';
  }

  async function open(token: AccessToken) {
    await TestBed.inject(Router).navigateByUrl(`/settings/access-tokens/${token.id}`);
    harness.detectChanges();
    httpTesting.expectOne('/api/v1/settings/tokens').flush([legacyToken, sibling]);
    await harness.fixture.whenStable();
    return rendered();
  }

  it('opens a token that has neither secrets nor an explicit role', async () => {
    expect(await open(legacyToken)).toContain('Secure token');
  });

  it('shows the token a create handed over', async () => {
    TestBed.inject(CreatedAccessTokenStore).put(created);
    expect(await open(legacyToken)).toContain(created.key);
  });

  it('does not carry the token over to a sibling', async () => {
    TestBed.inject(CreatedAccessTokenStore).put(created);
    await open(legacyToken);

    await TestBed.inject(Router).navigateByUrl(`/settings/access-tokens/${sibling.id}`);
    await harness.fixture.whenStable();

    expect(rendered()).not.toContain(created.key);
  });
});
