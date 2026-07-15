import {ChangeDetectionStrategy, Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowRight, faUsers} from '@fortawesome/free-solid-svg-icons';
import {of, startWith, Subject, switchMap} from 'rxjs';
import {organizationKind} from '../../../../util/organization-kind';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {UsersService} from '../../../services/users.service';
import {UsersComponent} from '../users.component';

@Component({
  templateUrl: './vendor-users.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [UsersComponent, RouterLink, FaIconComponent],
})
export class VendorUsersComponent {
  protected readonly faUsers = faUsers;
  protected readonly faArrowRight = faArrowRight;

  private readonly usersService = inject(UsersService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly auth = inject(AuthService);
  protected readonly refresh$ = new Subject<void>();

  private readonly allUsers = toSignal(
    this.refresh$.pipe(
      startWith(undefined),
      switchMap(() => this.usersService.getUsers())
    )
  );

  protected readonly users = computed(() => {
    const all = this.allUsers() ?? [];
    if (this.auth.isVendor()) {
      return all.filter((user) => organizationKind(user) === 'vendor');
    } else if (this.auth.isPartner()) {
      const partnerOrgId = this.auth.getPartnerOrganizationId();
      return all.filter((user) => user.partnerOrganizationId === partnerOrgId);
    }
    return all;
  });

  protected readonly customerOrganizations = toSignal(
    this.auth.isVendor() ? this.customerOrganizationsService.getCustomerOrganizations() : of([])
  );
}
