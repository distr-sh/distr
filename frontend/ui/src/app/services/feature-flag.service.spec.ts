import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {of} from 'rxjs';
import {FeatureFlagService} from './feature-flag.service';
import {OrganizationService} from './organization.service';

describe('FeatureFlagService', () => {
  let http: HttpTestingController;
  let service: FeatureFlagService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: OrganizationService,
          useValue: {
            get: () => of({features: [], subscriptionType: 'community'}),
          },
        },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    service = TestBed.inject(FeatureFlagService);
  });

  afterEach(() => {
    http.verify();
  });

  it('loads experimental feature flags from the admin endpoint', () => {
    service.getExperimentalFeatureFlags().subscribe((flags) => {
      expect(flags).toEqual([
        {
          key: 'environments',
          label: 'Environments',
          description: 'Groups deployment targets by promotion stage or operational purpose.',
          milestone: 'Milestone B',
          enabled: true,
        },
      ]);
    });

    const req = http.expectOne('/api/v1/experimental-feature-flags');
    expect(req.request.method).toBe('GET');
    req.flush([
      {
        key: 'environments',
        label: 'Environments',
        description: 'Groups deployment targets by promotion stage or operational purpose.',
        milestone: 'Milestone B',
        enabled: true,
      },
    ]);
  });
});
