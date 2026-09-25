import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {of} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {OverlayService} from '../services/overlay.service';
import {UpdateNotificationsComponent} from './update-notifications.component';

describe('UpdateNotificationsComponent', () => {
  let httpTesting: HttpTestingController;
  let fixture: ComponentFixture<UpdateNotificationsComponent>;

  beforeEach(async () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {hasAnyRole: () => true, isVendor: () => true, isCustomer: () => false, getClaims: () => ({})},
        },
        {
          provide: ContextService,
          useValue: {
            getCustomerOrganization: () => of(undefined),
            getUser: () => of({id: '22222222-2222-2222-2222-222222222222', email: 'a@b.c', userRole: 'admin'}),
          },
        },
      ],
    });
    // OverlayService needs a ViewContainerRef, so it is provided where a component injector exists.
    // The real one is used on purpose: the drawer is a template that only this path renders, and
    // everything it needs has to be reachable from the component's injector.
    TestBed.overrideComponent(UpdateNotificationsComponent, {
      add: {providers: [OverlayService]},
    });

    httpTesting = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(UpdateNotificationsComponent);
    fixture.detectChanges();
    httpTesting.expectOne('/api/v1/update-notification-configurations').flush([]);
    // Everything the drawer shows is prefetched with the page, the customer organizations that
    // group the recipients included.
    flushPicklists();
    httpTesting.expectOne('/api/v1/customer-organizations').flush([]);
    await fixture.whenStable();
    fixture.detectChanges();
  });

  afterEach(() => {
    fixture.destroy();
    httpTesting.verify();
  });

  function flushPicklists() {
    httpTesting
      .expectOne('/api/v1/applications')
      .flush([{id: '11111111-1111-1111-1111-111111111111', name: 'alpha-app', type: 'docker', versions: []}]);
    httpTesting
      .expectOne('/api/v1/artifacts')
      .flush([{id: '33333333-3333-3333-3333-333333333333', name: 'beta-artifact'}]);
    httpTesting
      .expectOne('/api/v1/user-accounts')
      .flush([{id: '22222222-2222-2222-2222-222222222222', email: 'a@b.c', name: 'Aaa', userRole: 'admin'}]);
  }

  async function settle() {
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  }

  function buttonWithText(text: string): HTMLButtonElement {
    const buttons = Array.from(document.body.querySelectorAll('button') as NodeListOf<HTMLButtonElement>);
    return buttons.find((button) => button.textContent?.trim() === text)!;
  }

  it('opens the drawer from the cache and switches it between applications and artifacts', async () => {
    buttonWithText('Update Notification').click();
    await settle();

    // The drawer is rendered before the refresh behind it has answered, and the recipient list
    // needs no request of its own.
    expect(document.body.textContent).toContain('Users to notify');
    flushPicklists();
    await settle();

    expect(document.body.textContent).toContain('Users to notify');
    expect(document.body.textContent).toContain('alpha-app');
    expect(document.body.textContent).toContain('Aaa');
    expect(document.body.textContent).not.toContain('beta-artifact');

    buttonWithText('Artifacts').click();
    await settle();

    expect(document.body.textContent).toContain('beta-artifact');
    expect(document.body.textContent).not.toContain('alpha-app');
  });

  it('saves one configuration that watches an application and an artifact', async () => {
    buttonWithText('Update Notification').click();
    await settle();
    flushPicklists();
    await settle();

    const nameInput = document.body.querySelector('#name') as HTMLInputElement;
    nameInput.value = 'everything we ship';
    nameInput.dispatchEvent(new Event('input'));
    expect(tabCount('application')).toBeUndefined();
    checkboxBefore('alpha-app').click();
    checkboxBefore('Aaa').click();
    await settle();
    expect(tabCount('application')).toBe('1');

    buttonWithText('Artifacts').click();
    await settle();
    checkboxBefore('beta-artifact').click();
    await settle();
    expect(tabCount('artifact')).toBe('1');

    const save = buttonWithText('Save');
    expect(save.disabled).toBe(false);
    save.click();
    await settle();

    const request = httpTesting.expectOne(
      (it) => it.method === 'POST' && it.url === '/api/v1/update-notification-configurations'
    );
    expect(request.request.body).toEqual({
      name: 'everything we ship',
      enabled: true,
      applicationIds: ['11111111-1111-1111-1111-111111111111'],
      artifactIds: ['33333333-3333-3333-3333-333333333333'],
      userAccountIds: ['22222222-2222-2222-2222-222222222222'],
    });
    request.flush({});
    await settle();
    httpTesting.expectOne('/api/v1/update-notification-configurations').flush([]);
  });
});

describe('UpdateNotificationsComponent scoped to a customer', () => {
  const customerOrganizationId = '44444444-4444-4444-4444-444444444444';
  const entitledApplicationId = '11111111-1111-1111-1111-111111111111';
  const customerUserId = '55555555-5555-5555-5555-555555555555';
  let httpTesting: HttpTestingController;
  let fixture: ComponentFixture<UpdateNotificationsComponent>;

  beforeEach(async () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {hasAnyRole: () => true, isVendor: () => true, isCustomer: () => false, getClaims: () => ({})},
        },
        {
          provide: ContextService,
          useValue: {
            getCustomerOrganization: () => of(undefined),
            getUser: () => of({id: '22222222-2222-2222-2222-222222222222', email: 'a@b.c', userRole: 'admin'}),
          },
        },
      ],
    });
    TestBed.overrideComponent(UpdateNotificationsComponent, {add: {providers: [OverlayService]}});

    httpTesting = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(UpdateNotificationsComponent);
    fixture.componentRef.setInput('customerOrganizationId', customerOrganizationId);
    fixture.detectChanges();

    httpTesting
      .expectOne('/api/v1/customer-organizations')
      .flush([{id: customerOrganizationId, name: 'Customer', features: ['deployment_targets', 'artifacts']}]);
    await fixture.whenStable();
    fixture.detectChanges();
  });

  afterEach(() => {
    fixture.destroy();
    httpTesting.verify();
  });

  it('configures what the customer is entitled to and notifies only its users', async () => {
    httpTesting
      .expectOne(`/api/v1/update-notification-configurations?customerOrganizationId=${customerOrganizationId}`)
      .flush([]);
    httpTesting.expectOne('/api/v1/application-entitlements').flush([
      {id: 'e1', name: 'entitled', customerOrganizationId, applicationId: entitledApplicationId},
      {
        id: 'e2',
        name: 'other customer',
        customerOrganizationId: 'another-customer',
        applicationId: '99999999-9999-9999-9999-999999999999',
      },
    ]);
    httpTesting.expectOne('/api/v1/artifact-entitlements').flush([]);
    httpTesting.expectOne('/api/v1/applications').flush([
      {id: entitledApplicationId, name: 'alpha-app', type: 'docker', versions: []},
      {id: '99999999-9999-9999-9999-999999999999', name: 'other-app', type: 'docker', versions: []},
    ]);
    httpTesting.expectOne('/api/v1/artifacts').flush([]);
    httpTesting.expectOne('/api/v1/user-accounts').flush([
      {id: '22222222-2222-2222-2222-222222222222', email: 'vendor@b.c', name: 'Vendor User', userRole: 'admin'},
      {id: customerUserId, email: 'customer@b.c', name: 'Customer User', userRole: 'admin', customerOrganizationId},
    ]);
    await fixture.whenStable();
    fixture.detectChanges();

    buttonWithText('Update Notification').click();
    await fixture.whenStable();
    fixture.detectChanges();
    // The refresh behind the drawer answers with what the page already has.
    httpTesting
      .expectOne('/api/v1/applications')
      .flush([{id: entitledApplicationId, name: 'alpha-app', type: 'docker', versions: []}]);
    httpTesting.expectOne('/api/v1/artifacts').flush([]);
    httpTesting
      .expectOne('/api/v1/user-accounts')
      .flush([
        {id: customerUserId, email: 'customer@b.c', name: 'Customer User', userRole: 'admin', customerOrganizationId},
      ]);
    await fixture.whenStable();
    fixture.detectChanges();

    expect(document.body.textContent).toContain('alpha-app');
    expect(document.body.textContent).not.toContain('other-app');
    expect(document.body.textContent).toContain('Customer User');
    expect(document.body.textContent).not.toContain('Vendor User');

    const nameInput = document.body.querySelector('#name') as HTMLInputElement;
    nameInput.value = 'for the customer';
    nameInput.dispatchEvent(new Event('input'));
    checkboxBefore('alpha-app').click();
    checkboxBefore('Customer User').click();
    await fixture.whenStable();
    fixture.detectChanges();

    buttonWithText('Save').click();
    await fixture.whenStable();
    fixture.detectChanges();

    const request = httpTesting.expectOne(
      (it) => it.method === 'POST' && it.url === '/api/v1/update-notification-configurations'
    );
    expect(request.request.body).toEqual({
      customerOrganizationId,
      name: 'for the customer',
      enabled: true,
      applicationIds: [entitledApplicationId],
      artifactIds: [],
      userAccountIds: [customerUserId],
    });
    request.flush({});
    await fixture.whenStable();
    fixture.detectChanges();

    httpTesting
      .expectOne(`/api/v1/update-notification-configurations?customerOrganizationId=${customerOrganizationId}`)
      .flush([]);
  });

  function buttonWithText(text: string): HTMLButtonElement {
    const buttons = Array.from(document.body.querySelectorAll('button') as NodeListOf<HTMLButtonElement>);
    return buttons.find((button) => button.textContent?.trim() === text)!;
  }
});

/** The number of selected entries the given picklist tab shows, if any. */
function tabCount(tabId: string): string | undefined {
  return document.body.querySelector(`#tab-${tabId} .distr-tag-badge`)?.textContent?.trim();
}

/** The checkbox of the row whose label contains the given text. */
function checkboxBefore(text: string): HTMLInputElement {
  const rows = Array.from(document.body.querySelectorAll('label') as NodeListOf<HTMLLabelElement>);
  const row = rows.find((label) => label.textContent?.includes(text))!;
  return row.querySelector('input[type=checkbox]')!;
}
