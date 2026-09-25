import {provideHttpClient} from '@angular/common/http';
import {provideHttpClientTesting} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {AuthService} from './auth.service';

function jwt(claims: Record<string, unknown>): string {
  const encode = (o: unknown) => btoa(JSON.stringify(o)).replaceAll('+', '-').replaceAll('/', '_').replace(/=+$/, '');
  return `${encode({alg: 'HS256', typ: 'JWT'})}.${encode(claims)}.signature`;
}

const vendorAdmin = jwt({sub: 'a', email: 'admin@acme.test', org: 'acme', role: 'admin'});
const customerReadOnly = jwt({sub: 'b', email: 'user@customer.test', org: 'acme', c_org: 'cust', role: 'customer'});

// The test environment provides no Web Storage, so the parts of it that AuthService uses are stubbed out.
function memoryStorage(): Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> {
  const entries = new Map<string, string>();
  return {
    getItem: (key) => entries.get(key) ?? null,
    setItem: (key, value) => void entries.set(key, value),
    removeItem: (key) => void entries.delete(key),
  };
}

describe('AuthService', () => {
  let service: AuthService;

  beforeEach(() => {
    vi.stubGlobal('localStorage', memoryStorage());
    vi.stubGlobal('sessionStorage', memoryStorage());
    TestBed.configureTestingModule({providers: [provideHttpClient(), provideHttpClientTesting()]});
    service = TestBed.inject(AuthService);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // The decoded claims are cached because every role and context check goes through them. A cache that outlives
  // the token it was decoded from would keep granting the permissions of the previous session.
  it('reflects the new claims after the token is replaced', () => {
    service.loginWithToken(vendorAdmin);
    expect(service.isVendor()).toBe(true);
    expect(service.hasAnyRole('read_write', 'admin')).toBe(true);

    service.loginWithToken(customerReadOnly);

    expect(service.isVendor()).toBe(false);
    expect(service.isCustomer()).toBe(true);
    expect(service.hasAnyRole('read_write', 'admin')).toBe(false);
  });

  it('prefers the action token over the session token', () => {
    service.loginWithToken(vendorAdmin);
    expect(service.getClaims()?.email).toBe('admin@acme.test');

    service.actionToken = jwt({sub: 'c', email: 'invitee@acme.test', scope: 'invite'});

    expect(service.getClaims()?.email).toBe('invitee@acme.test');
    expect(service.isLoggedIn()).toBe(false);
  });
});
