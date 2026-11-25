import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {refreshInBackground} from '../../../../util/refresh';
import {UsersService} from '../../../services/users.service';
import {UsersComponent} from '../../users/users.component';

@Component({
  templateUrl: './customer-users-tab.component.html',
  imports: [UsersComponent],
})
export class CustomerUsersTabComponent {
  private readonly usersService = inject(UsersService);

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );

  private readonly allUsers = toSignal(this.usersService.getUsers());
  protected readonly users = computed(() =>
    this.allUsers()?.filter((it) => it.customerOrganizationId === this.customerOrganizationId())
  );

  constructor() {
    refreshInBackground(this.usersService.refresh());
  }
}
