import {OverlayModule} from '@angular/cdk/overlay';
import {Component, computed, ElementRef, inject, signal, viewChild} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faBoxesStacked, faChevronDown} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, map, startWith, Subject, switchMap} from 'rxjs';
import {ServiceAccountsComponent} from '../../../service-accounts/service-accounts.component';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {UsersService} from '../../../services/users.service';
import {UsersComponent} from '../users.component';

@Component({
  templateUrl: './customer-users.component.html',
  imports: [UsersComponent, ServiceAccountsComponent, RouterLink, FontAwesomeModule, OverlayModule],
})
export class CustomerUsersComponent {
  protected readonly auth = inject(AuthService);
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faChevronDown = faChevronDown;

  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly usersService = inject(UsersService);
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );
  protected readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  protected readonly customerOrganization = computed(() => {
    const id = this.customerOrganizationId();
    return this.customerOrganizations()?.find((org) => org.id === id);
  });

  protected readonly refresh$ = new Subject<void>();
  protected readonly users = toSignal(
    combineLatest([
      this.refresh$.pipe(
        startWith(undefined),
        switchMap(() => this.usersService.getUsers())
      ),
      toObservable(this.customerOrganizationId),
    ]).pipe(
      map(([users, customerOrganizationId]) =>
        users.filter((it) => it.customerOrganizationId === customerOrganizationId)
      )
    )
  );

  protected readonly dropdownTriggerButton = viewChild.required<ElementRef<HTMLElement>>('dropdownTriggerButton');
  protected readonly breadcrumbDropdown = signal(false);
  breadcrumbDropdownWidth = 0;

  protected toggleBreadcrumbDropdown() {
    this.breadcrumbDropdown.update((v) => !v);
    if (this.breadcrumbDropdown()) {
      this.breadcrumbDropdownWidth = this.dropdownTriggerButton().nativeElement.getBoundingClientRect().width;
    }
  }
}
