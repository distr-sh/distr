import {provideHttpClient} from '@angular/common/http';
import {HttpTestingController, provideHttpClientTesting} from '@angular/common/http/testing';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AuthService} from '../services/auth.service';
import {OverlayService} from '../services/overlay.service';
import {ApplicationNotificationConfigurationsComponent} from './application-notification-configurations.component';

describe('ApplicationNotificationConfigurationsComponent', () => {
  let httpTesting: HttpTestingController;
  let fixture: ComponentFixture<ApplicationNotificationConfigurationsComponent>;

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
      ],
    });
    // OverlayService needs a ViewContainerRef, so it is provided where a component injector exists.
    // The real one is used on purpose: the drawer is a template that only this path renders, and
    // everything it needs has to be reachable from the component's injector.
    TestBed.overrideComponent(ApplicationNotificationConfigurationsComponent, {
      add: {providers: [OverlayService]},
    });

    httpTesting = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(ApplicationNotificationConfigurationsComponent);
    fixture.detectChanges();
    httpTesting.expectOne('/api/v1/application-notification-configurations').flush([]);
    httpTesting
      .expectOne('/api/v1/applications')
      .flush([{id: '11111111-1111-1111-1111-111111111111', name: 'app', type: 'docker', versions: []}]);
    httpTesting
      .expectOne('/api/v1/user-accounts')
      .flush([{id: '22222222-2222-2222-2222-222222222222', email: 'a@b.c', name: 'Aaa', userRole: 'admin'}]);
    await fixture.whenStable();
    fixture.detectChanges();
  });

  afterEach(() => {
    fixture.destroy();
    httpTesting.verify();
  });

  it('opens the drawer with the applications and recipients to pick from', async () => {
    const create = Array.from(fixture.nativeElement.querySelectorAll('button') as NodeListOf<HTMLButtonElement>).find(
      (button) => button.textContent?.includes('Create Notification')
    );
    create!.click();
    httpTesting.expectOne('/api/v1/customer-organizations').flush([]);
    await fixture.whenStable();
    fixture.detectChanges();

    const drawer = document.body.textContent ?? '';
    expect(drawer).toContain('New Application Notification');
    expect(drawer).toContain('app');
    expect(drawer).toContain('Aaa');
  });
});
