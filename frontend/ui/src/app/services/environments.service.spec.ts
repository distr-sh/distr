import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {EnvironmentsService} from './environments.service';

describe('EnvironmentsService', () => {
  let http: HttpTestingController;
  let service: EnvironmentsService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    service = TestBed.inject(EnvironmentsService);
  });

  afterEach(() => {
    http.verify();
  });

  it('lists environments from the environments endpoint', () => {
    service.list().subscribe((environments) => {
      expect(environments).toEqual([
        {
          id: 'env-1',
          createdAt: '2026-06-20T09:30:00Z',
          updatedAt: '2026-06-20T10:45:00Z',
          name: 'Production',
          description: 'Customer production targets',
          sortOrder: 30,
          isProduction: true,
          allowDynamicTargets: false,
        },
      ]);
    });

    const req = http.expectOne('/api/v1/environments');
    expect(req.request.method).toBe('GET');
    req.flush([
      {
        id: 'env-1',
        createdAt: '2026-06-20T09:30:00Z',
        updatedAt: '2026-06-20T10:45:00Z',
        name: 'Production',
        description: 'Customer production targets',
        sortOrder: 30,
        isProduction: true,
        allowDynamicTargets: false,
      },
    ]);
  });

  it('creates, updates, and deletes environments', () => {
    service.create({name: 'Development', description: '', sortOrder: 10}).subscribe();
    const createReq = http.expectOne('/api/v1/environments');
    expect(createReq.request.method).toBe('POST');
    expect(createReq.request.body).toEqual({name: 'Development', description: '', sortOrder: 10});
    createReq.flush({
      id: 'env-2',
      createdAt: '2026-06-20T09:30:00Z',
      updatedAt: '2026-06-20T09:30:00Z',
      name: 'Development',
      description: '',
      sortOrder: 10,
      isProduction: false,
      allowDynamicTargets: false,
    });

    service
      .update('env-2', {
        name: 'Dev',
        description: 'Shared development targets',
        sortOrder: 20,
        isProduction: false,
        allowDynamicTargets: true,
      })
      .subscribe();
    const updateReq = http.expectOne('/api/v1/environments/env-2');
    expect(updateReq.request.method).toBe('PUT');
    expect(updateReq.request.body).toEqual({
      name: 'Dev',
      description: 'Shared development targets',
      sortOrder: 20,
      isProduction: false,
      allowDynamicTargets: true,
    });
    updateReq.flush({
      id: 'env-2',
      createdAt: '2026-06-20T09:30:00Z',
      updatedAt: '2026-06-20T10:45:00Z',
      name: 'Dev',
      description: 'Shared development targets',
      sortOrder: 20,
      isProduction: false,
      allowDynamicTargets: true,
    });

    service.delete('env-2').subscribe();
    const deleteReq = http.expectOne('/api/v1/environments/env-2');
    expect(deleteReq.request.method).toBe('DELETE');
    deleteReq.flush(null);
  });
});
