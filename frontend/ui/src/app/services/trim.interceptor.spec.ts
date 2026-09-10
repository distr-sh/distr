import {HttpClient, provideHttpClient, withInterceptors} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {skipTrim, trimInterceptor} from './trim.interceptor';

describe('trimInterceptor', () => {
  let httpTesting: HttpTestingController;
  let http: HttpClient;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(withInterceptors([trimInterceptor])), provideHttpClientTesting()],
    });
    httpTesting = TestBed.inject(HttpTestingController);
    http = TestBed.inject(HttpClient);
  });

  afterEach(() => httpTesting.verify());

  it('trims nested strings', () => {
    http
      .post('/api/v1/test', {
        name: '  Acme Inc. ',
        password: '  a password  ',
        tags: [' one ', 'two'],
        nested: {label: ' nested '},
        count: 3,
      })
      .subscribe();

    expect(httpTesting.expectOne('/api/v1/test').request.body).toEqual({
      name: 'Acme Inc.',
      password: 'a password',
      tags: ['one', 'two'],
      nested: {label: 'nested'},
      count: 3,
    });
  });

  it('leaves a FormData body untouched', () => {
    const body = new FormData();
    body.append('name', '  Acme Inc. ');
    http.post('/api/v1/test', body).subscribe();

    expect(httpTesting.expectOne('/api/v1/test').request.body).toBe(body);
  });

  it('leaves only the opted out keys untouched', () => {
    http.post('/api/v1/test', {key: '  a key ', value: '  a secret value\n'}, {context: skipTrim('value')}).subscribe();

    expect(httpTesting.expectOne('/api/v1/test').request.body).toEqual({
      key: 'a key',
      value: '  a secret value\n',
    });
  });
});
