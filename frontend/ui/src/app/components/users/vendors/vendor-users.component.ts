import {ChangeDetectionStrategy, Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {faUsers} from '@fortawesome/free-solid-svg-icons';
import {startWith, Subject, switchMap} from 'rxjs';
import {organizationKind} from '../../../../util/organization-kind';
import {AuthService} from '../../../services/auth.service';
import {UsersService} from '../../../services/users.service';
import {OrganizationScopeNavComponent} from '../../organization-scope-nav.component';
import {PageComponent} from '../../page.component';
import {UsersComponent} from '../users.component';

@Component({
  templateUrl: './vendor-users.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [UsersComponent, OrganizationScopeNavComponent, PageComponent],
})
export class VendorUsersComponent {
  protected readonly faUsers = faUsers;

  private readonly usersService = inject(UsersService);
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
}
