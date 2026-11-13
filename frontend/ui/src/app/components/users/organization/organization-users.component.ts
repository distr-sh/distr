import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {startWith, Subject, switchMap} from 'rxjs';
import {refreshInBackground} from '../../../../util/refresh';
import {AuthService} from '../../../services/auth.service';
import {UsersService} from '../../../services/users.service';
import {PageComponent} from '../../page.component';
import {UsersComponent} from '../users.component';

@Component({
  templateUrl: './organization-users.component.html',
  imports: [UsersComponent, PageComponent],
})
export class OrganizationUsersComponent {
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
    if (this.auth.isPartner()) {
      const partnerOrgId = this.auth.getPartnerOrganizationId();
      return all.filter((user) => user.partnerOrganizationId === partnerOrgId);
    }
    return all;
  });

  constructor() {
    refreshInBackground(this.usersService.refresh());
  }
}
